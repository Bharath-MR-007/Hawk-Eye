// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package hawkeye

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/internal/probes"
	"github.com/Bharath-MR-007/hawk-eye/internal/storage"
	"github.com/Bharath-MR-007/hawk-eye/pkg/api"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/runtime"
	"github.com/Bharath-MR-007/hawk-eye/pkg/config"
	"github.com/Bharath-MR-007/hawk-eye/pkg/db"
	"github.com/Bharath-MR-007/hawk-eye/pkg/hawkeye/metrics"
	"github.com/Bharath-MR-007/hawk-eye/pkg/hawkeye/targets"
	"gopkg.in/yaml.v3"
)

const shutdownTimeout = time.Second * 90

// Hawkeye is the main struct of the hawkeye application
type Hawkeye struct {
	// config is the startup configuration of the hawkeye
	config *config.Config
	// db is the database used to store the check results
	db db.DB
	// api is the hawkeye's API
	api api.API
	// loader is used to load the runtime configuration
	loader config.Loader
	// tarMan is the target manager that is used to manage global targets
	tarMan targets.TargetManager
	// metrics is used to collect metrics
	metrics metrics.Provider
	// controller is used to manage the checks
	controller *ChecksController
	// probeManager is used for advanced probing
	probeManager *probes.ProbeManager
	// tsDB is the time series database for drilldown analysis
	tsDB *storage.TimeSeriesDB
	// cRuntime is used to signal that the runtime configuration has changed
	cRuntime chan runtime.Config
	// cErr is used to handle non-recoverable errors of the hawkeye components
	cErr chan error
	// cDone is used to signal that the hawkeye was shut down because of an error
	cDone chan struct{}
	// shutOnce is used to ensure that the shutdown function is only called once
	shutOnce sync.Once
	// lastRuntimeConfig stores the most recent runtime configuration
	lastRuntimeConfig runtime.Config
	mu                sync.RWMutex
}

// New creates a new hawkeye from a given configfile
func New(cfg *config.Config) *Hawkeye {
	m := metrics.New(cfg.Telemetry)
	dbase := db.NewInMemory()

	hawkeye := &Hawkeye{
		config:       cfg,
		db:           dbase,
		api:          api.New(cfg.Api),
		metrics:      m,
		controller:   NewChecksController(dbase, m),
		probeManager: probes.NewProbeManager(),
		tsDB:         storage.NewTimeSeriesDB(dbase),
		cRuntime:     make(chan runtime.Config, 1),
		cErr:         make(chan error, 1),
		cDone:        make(chan struct{}, 1),
		shutOnce:     sync.Once{},
	}

	if cfg.HasTargetManager() {
		gm := targets.NewManager(cfg.HawkeyeName, cfg.TargetManager, m)
		hawkeye.tarMan = gm
	}
	hawkeye.loader = config.NewLoader(cfg, hawkeye.cRuntime)

	// Load local integration configs if they exist
	if b, err := os.ReadFile("nnmi_config.yaml"); err == nil {
		if err := yaml.Unmarshal(b, &hawkeye.config.Nnmi); err == nil {
			logger.FromContext(context.Background()).Info("Loaded NNMi configuration from nnmi_config.yaml")
		}
	}

	// Wire ProbeManager to ChecksController for live WebSocket streaming
	hawkeye.controller.WithProbeManager(hawkeye.probeManager)

	return hawkeye
}

// Run starts the hawkeye
func (s *Hawkeye) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	log := logger.FromContext(ctx)
	defer cancel()

	err := s.metrics.InitTracing(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	go func() {
		s.cErr <- s.loader.Run(ctx)
	}()
	go func() {
		if s.tarMan != nil {
			s.cErr <- s.tarMan.Reconcile(ctx)
		}
	}()

	go func() {
		s.cErr <- s.startupAPI(ctx)
	}()

	go func() {
		s.cErr <- s.controller.Run(ctx)
	}()

	for {
		select {
		case cfg := <-s.cRuntime:
			s.mu.Lock()
			s.lastRuntimeConfig = cfg
			s.mu.Unlock()
			cfg = s.enrichTargets(ctx, cfg)
			cfg = s.enrichIntegrations(ctx, cfg)
			s.controller.Reconcile(ctx, cfg)
		case <-ctx.Done():
			s.shutdown(ctx)
		case err := <-s.cErr:
			if err != nil {
				log.Error("Non-recoverable error in hawkeye component", "error", err)
				s.shutdown(ctx)
			}
		case <-s.cDone:
			log.InfoContext(ctx, "Hawkeye was shut down")
			return fmt.Errorf("hawkeye was shut down")
		}
	}
}

// enrichTargets updates the targets of the hawkeye's checks with the
// global targets. Per default, the two target lists are merged.
func (s *Hawkeye) enrichTargets(ctx context.Context, cfg runtime.Config) runtime.Config {
	l := logger.FromContext(ctx)
	if cfg.Empty() || s.tarMan == nil {
		return cfg
	}

	for _, gt := range s.tarMan.GetTargets() {
		u, err := url.Parse(gt.Url)
		if err != nil {
			l.Error("Failed to parse global target URL", "error", err, "url", gt.Url)
			continue
		}

		// split off hostWithoutPort because it could contain a port
		hostWithoutPort := strings.Split(u.Host, ":")[0]
		if hostWithoutPort == s.config.HawkeyeName {
			continue
		}

		if cfg.HasHealthCheck() && !slices.Contains(cfg.Health.Targets, u.String()) {
			cfg.Health.Targets = append(cfg.Health.Targets, u.String())
		}
		if cfg.HasLatencyCheck() && !slices.Contains(cfg.Latency.Targets, u.String()) {
			cfg.Latency.Targets = append(cfg.Latency.Targets, u.String())
		}
		if cfg.HasDNSCheck() && !slices.Contains(cfg.Dns.Targets, hostWithoutPort) {
			cfg.Dns.Targets = append(cfg.Dns.Targets, hostWithoutPort)
		}
	}

	return cfg
}

// enrichIntegrations ensures that global integration settings are applied to checks
func (s *Hawkeye) enrichIntegrations(ctx context.Context, cfg runtime.Config) runtime.Config {
	if cfg.Traceroute != nil && (cfg.Traceroute.Nnmi.Host == "" || !cfg.Traceroute.Nnmi.Enabled) {
		// Only override if check doesn't have it or if it's disabled in check but enabled globally
		if s.config.Nnmi.Enabled {
			cfg.Traceroute.Nnmi = s.config.Nnmi
		}
	}
	return cfg
}

// TriggerReconcile forces a reconciliation of the checks with the current configuration
func (s *Hawkeye) TriggerReconcile(ctx context.Context) {
	s.mu.RLock()
	cfg := s.lastRuntimeConfig
	s.mu.RUnlock()

	if !cfg.Empty() {
		cfg = s.enrichTargets(ctx, cfg)
		cfg = s.enrichIntegrations(ctx, cfg)
		s.controller.Reconcile(ctx, cfg)
	}
}

// shutdown shuts down the hawkeye and all managed components gracefully.
// It returns an error if one is present in the context or if any of the
// components fail to shut down.
func (s *Hawkeye) shutdown(ctx context.Context) {
	errC := ctx.Err()
	log := logger.FromContext(ctx)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	s.shutOnce.Do(func() {
		log.InfoContext(ctx, "Shutting down hawkeye")
		var sErrs ErrShutdown
		if s.tarMan != nil {
			sErrs.errTarMan = s.tarMan.Shutdown(ctx)
		}
		sErrs.errAPI = s.api.Shutdown(ctx)
		sErrs.errMetrics = s.metrics.Shutdown(ctx)
		s.loader.Shutdown(ctx)
		s.controller.Shutdown(ctx)

		if sErrs.HasError() {
			log.ErrorContext(ctx, "Failed to shutdown gracefully", "contextError", errC, "errors", sErrs)
		}

		// Signal that shutdown is complete
		s.cDone <- struct{}{}
	})
}
