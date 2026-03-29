// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package hawkeye

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"os"

	"github.com/go-chi/chi/v5"
	"github.com/Bharath-MR-007/hawk-eye/api/handlers"
	ws "github.com/Bharath-MR-007/hawk-eye/api/websocket"
	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/api"
	"gopkg.in/yaml.v3"
)

type encoder interface {
	Encode(v any) error
}

const urlParamCheckName = "checkName"

func (s *Hawkeye) startupAPI(ctx context.Context) error {
	routes := []api.Route{
		{
			Path: "/openapi", Method: http.MethodGet,
			Handler: s.handleOpenAPI,
		},
		{
			Path: fmt.Sprintf("/v1/metrics/{%s}", urlParamCheckName), Method: http.MethodGet,
			Handler: s.handleCheckMetrics,
		},
		{
			Path: "/metrics", Method: "*",
			Handler: promhttp.HandlerFor(
				s.metrics.GetRegistry(),
				promhttp.HandlerOpts{Registry: s.metrics.GetRegistry()},
			).ServeHTTP,
		},
	}

	drilldown := handlers.NewDrillDownHandler(s.probeManager, s.tsDB, s.controller, s.db)
	routes = append(routes, drilldown.GetRoutes()...)

	wsHandler := ws.NewHandler(s.probeManager)

	// Auth check middleware
	withAuth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("hawk_session")
			if err != nil || cookie.Value != "authenticated" {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			h(w, r)
		}
	}

	routes = append(routes, []api.Route{
		{Path: "/login", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/login.html"); err == nil {
				http.ServeFile(w, r, "/login.html")
				return
			}
			http.ServeFile(w, r, "login.html")
		}},
		{Path: "/logout", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:   "hawk_session",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/login", http.StatusFound)
		}},
		{Path: "/api/v1/targets", Method: http.MethodPost, Handler: withAuth(s.handleAddTarget)},
		{Path: "/api/v1/targets", Method: http.MethodDelete, Handler: withAuth(s.handleDeleteTarget)},
		{Path: "/api/v1/alerts", Method: http.MethodGet, Handler: withAuth(s.handleGetAlerts)},
		{Path: "/api/v1/alerts", Method: http.MethodPost, Handler: withAuth(s.handleUpdateAlerts)},
		{Path: "/api/v1/config/export", Method: http.MethodPost, Handler: withAuth(s.handleExportConfig)},
		{Path: "/api/v1/config/import", Method: http.MethodPost, Handler: withAuth(s.handleImportConfig)},
		{Path: "/api/v1/polling", Method: http.MethodGet, Handler: withAuth(s.handleGetPolling)},
		{Path: "/api/v1/polling", Method: http.MethodPost, Handler: withAuth(s.handleUpdatePolling)},
		{Path: "/api/v1/config/snmp", Method: http.MethodGet, Handler: withAuth(s.handleGetSnmp)},
		{Path: "/api/v1/config/snmp", Method: http.MethodPost, Handler: withAuth(s.handleUpdateSnmp)},
		{Path: "/api/v1/notifications/snmp", Method: http.MethodPost, Handler: s.handleSnmpTrap},
		{Path: "/api/v1/config/nnmi", Method: http.MethodGet, Handler: withAuth(s.handleGetNnmi)},
		{Path: "/api/v1/config/nnmi", Method: http.MethodPost, Handler: withAuth(s.handleUpdateNnmi)},
		{Path: "/api/v1/config/nnmi/test", Method: http.MethodPost, Handler: withAuth(s.handleTestNnmi)},
		{Path: "/api/v1/reachability/test", Method: http.MethodPost, Handler: withAuth(s.handleReachabilityTest)},
		{Path: "/api/v1/ws/live", Method: http.MethodGet, Handler: wsHandler.HandleLiveUpdates},
		{Path: "/scripts/{filename}", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
			filename := chi.URLParam(r, "filename")
			http.ServeFile(w, r, "scripts/"+filename)
		}},
		{Path: "/", Method: http.MethodGet, Handler: func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		}},
		{Path: "/dashboard", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/dashboard.html"); err == nil {
				http.ServeFile(w, r, "/dashboard.html")
				return
			}
			http.ServeFile(w, r, "dashboard.html")
		})},
		{Path: "/live_dashboard", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/live_dashboard.html"); err == nil {
				http.ServeFile(w, r, "/live_dashboard.html")
				return
			}
			http.ServeFile(w, r, "live_dashboard.html")
		})},
		{Path: "/inventory", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/inventory.html"); err == nil {
				http.ServeFile(w, r, "/inventory.html")
				return
			}
			http.ServeFile(w, r, "inventory.html")
		})},
		{Path: "/incidents", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/incidents.html"); err == nil {
				http.ServeFile(w, r, "/incidents.html")
				return
			}
			http.ServeFile(w, r, "incidents.html")
		})},
		{Path: "/integrations", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/integrations.html"); err == nil {
				http.ServeFile(w, r, "/integrations.html")
				return
			}
			http.ServeFile(w, r, "integrations.html")
		})},
		{Path: "/integrations_config", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/integrations_config.html"); err == nil {
				http.ServeFile(w, r, "/integrations_config.html")
				return
			}
			http.ServeFile(w, r, "integrations_config.html")
		})},
		{Path: "/integrations_guide", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/integrations_guide.html"); err == nil {
				http.ServeFile(w, r, "/integrations_guide.html")
				return
			}
			http.ServeFile(w, r, "integrations_guide.html")
		})},
		{Path: "/alerts", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/alerts_config.html"); err == nil {
				http.ServeFile(w, r, "/alerts_config.html")
				return
			}
			http.ServeFile(w, r, "alerts_config.html")
		})},
		{Path: "/polling", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/polling_config.html"); err == nil {
				http.ServeFile(w, r, "/polling_config.html")
				return
			}
			http.ServeFile(w, r, "polling_config.html")
		})},
		{Path: "/installation.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/installation.html"); err == nil {
				http.ServeFile(w, r, "/installation.html")
				return
			}
			http.ServeFile(w, r, "installation.html")
		})},
		{Path: "/requirements.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/requirements.html"); err == nil {
				http.ServeFile(w, r, "/requirements.html")
				return
			}
			http.ServeFile(w, r, "requirements.html")
		})},
		{Path: "/capabilities.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/capabilities.html"); err == nil {
				http.ServeFile(w, r, "/capabilities.html")
				return
			}
			http.ServeFile(w, r, "capabilities.html")
		})},
		{Path: "/usermanual.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/usermanual.html"); err == nil {
				http.ServeFile(w, r, "/usermanual.html")
				return
			}
			http.ServeFile(w, r, "usermanual.html")
		})},
		{Path: "/admindoc.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/admindoc.html"); err == nil {
				http.ServeFile(w, r, "/admindoc.html")
				return
			}
			http.ServeFile(w, r, "admindoc.html")
		})},
		{Path: "/about.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/about.html"); err == nil {
				http.ServeFile(w, r, "/about.html")
				return
			}
			http.ServeFile(w, r, "about.html")
		})},
		{Path: "/architecture.html", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/architecture.html"); err == nil {
				http.ServeFile(w, r, "/architecture.html")
				return
			}
			http.ServeFile(w, r, "architecture.html")
		})},
		{Path: "/users_config", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/users_config.html"); err == nil {
				http.ServeFile(w, r, "/users_config.html")
				return
			}
			http.ServeFile(w, r, "users_config.html")
		})},
		{Path: "/target_detail", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/target_detail.html"); err == nil {
				http.ServeFile(w, r, "/target_detail.html")
				return
			}
			http.ServeFile(w, r, "target_detail.html")
		})},
		{Path: "/troubleshooting", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/troubleshooting.html"); err == nil {
				http.ServeFile(w, r, "/troubleshooting.html")
				return
			}
			http.ServeFile(w, r, "troubleshooting.html")
		})},
		{Path: "/device_reachability", Method: http.MethodGet, Handler: withAuth(func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("/device_reachability.html"); err == nil {
				http.ServeFile(w, r, "/device_reachability.html")
				return
			}
			http.ServeFile(w, r, "device_reachability.html")
		})},
	}...)

	err := s.api.RegisterRoutes(ctx, routes...)
	if err != nil {
		logger.FromContext(ctx).Error("Error while registering routes", "error", err)
		return err
	}
	return s.api.Run(ctx)
}

func (s *Hawkeye) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	oapi, err := s.controller.GenerateCheckSpecs(r.Context())
	if err != nil {
		log.Error("failed to create openapi", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}

	mime := r.Header.Get("Accept")

	var marshaler encoder
	switch mime {
	case "application/json":
		marshaler = json.NewEncoder(w)
		w.Header().Add("Content-Type", "application/json")
	default:
		marshaler = yaml.NewEncoder(w)
		w.Header().Add("Content-Type", "text/yaml")
	}

	err = marshaler.Encode(oapi)
	if err != nil {
		log.Error("failed to marshal openapi", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
}

func (s *Hawkeye) handleCheckMetrics(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	name := chi.URLParam(r, urlParamCheckName)
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
	res, ok := s.db.Get(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(http.StatusText(http.StatusNotFound)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(res); err != nil {
		log.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		if err != nil {
			log.Error("Failed to write response", "error", err)
		}
		return
	}
	w.Header().Add("Content-Type", "application/json")
}
