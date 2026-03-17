package hawkeye

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
	"github.com/Bharath-MR-007/hawk-eye/pkg/checks/runtime"
	"gopkg.in/yaml.v3"
)

type AddTargetRequest struct {
	URL string `json:"url"`
}

func (s *Hawkeye) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req AddTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(req.URL)
	if err != nil || u.Host == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	host := u.Host
	if u.Port() != "" {
		host = u.Hostname()
	}

	configPath := s.config.Loader.File.Path
	if configPath == "" {
		configPath = "checks.yaml"
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("Failed to read config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	var m yaml.Node
	if err := yaml.Unmarshal(b, &m); err != nil {
		log.Error("Failed to parse config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Helper to create scalar node
	scalar := func(v string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v}
	}

	// Helper to find key in a mapping node
	findKey := func(node *yaml.Node, key string) *yaml.Node {
		if node.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return node.Content[i+1]
			}
		}
		return nil
	}

	for _, doc := range m.Content {
		if healthNode := findKey(doc, "health"); healthNode != nil {
			if targetsNode := findKey(healthNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				targetsNode.Content = append(targetsNode.Content, scalar(req.URL))
			}
		}

		// Latency
		if latencyNode := findKey(doc, "latency"); latencyNode != nil {
			if targetsNode := findKey(latencyNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				targetsNode.Content = append(targetsNode.Content, scalar(req.URL))
			}
		}

		// Http Advanced
		if httpAdvNode := findKey(doc, "http_advanced"); httpAdvNode != nil {
			if targetsNode := findKey(httpAdvNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				mapNode := &yaml.Node{Kind: yaml.MappingNode}
				mapNode.Content = append(mapNode.Content,
					scalar("url"), scalar(req.URL),
					scalar("method"), scalar("GET"),
					scalar("expected_status"), scalar("200"),
				)
				targetsNode.Content = append(targetsNode.Content, mapNode)
			}
		}

		// SSL TLS
		if sslNode := findKey(doc, "ssl_tls"); sslNode != nil {
			if targetsNode := findKey(sslNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				targetsNode.Content = append(targetsNode.Content, scalar(host+":443"))
			}
		}

		// DNS Advanced
		if dnsNode := findKey(doc, "dns_advanced"); dnsNode != nil {
			if queriesNode := findKey(dnsNode, "queries"); queriesNode != nil && queriesNode.Kind == yaml.SequenceNode {
				mapNode := &yaml.Node{Kind: yaml.MappingNode}
				mapNode.Content = append(mapNode.Content,
					scalar("name"), scalar(host),
					scalar("type"), scalar("A"),
				)
				queriesNode.Content = append(queriesNode.Content, mapNode)
			}
		}

		// TCP Metrics
		if tcpNode := findKey(doc, "tcp_metrics"); tcpNode != nil {
			if targetsNode := findKey(tcpNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				targetsNode.Content = append(targetsNode.Content, scalar(host+":443"))
			}
		}

		// Traceroute
		if traceNode := findKey(doc, "traceroute"); traceNode != nil {
			if targetsNode := findKey(traceNode, "targets"); targetsNode != nil && targetsNode.Kind == yaml.SequenceNode {
				mapNode := &yaml.Node{Kind: yaml.MappingNode}
				mapNode.Content = append(mapNode.Content,
					scalar("addr"), scalar(host),
					scalar("port"), scalar("80"),
				)
				targetsNode.Content = append(targetsNode.Content, mapNode)
			}
		}
	}

	out, err := yaml.Marshal(&m)
	if err != nil {
		log.Error("Failed to marshal config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		log.Error("Failed to write config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Trigger immediate reload
	var newCfg runtime.Config
	if err := yaml.Unmarshal(out, &newCfg); err == nil {
		s.cRuntime <- newCfg
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

type DeleteTargetRequest struct {
	URL string `json:"url"`
}

func (s *Hawkeye) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req DeleteTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(req.URL)
	if err != nil || u.Host == "" {
		hostOnly := req.URL
		u, err = url.Parse("https://" + hostOnly)
		if err != nil || u.Host == "" {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
	}

	host := u.Host
	if u.Port() != "" {
		host = u.Hostname()
	}

	log.Info("Deleting target", "input", req.URL, "parsedHost", host)

	configPath := s.config.Loader.File.Path
	if configPath == "" {
		configPath = "checks.yaml"
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("Failed to read config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	var m yaml.Node
	if err := yaml.Unmarshal(b, &m); err != nil {
		log.Error("Failed to parse config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Helper to find key in a mapping node
	findKey := func(node *yaml.Node, key string) *yaml.Node {
		if node.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return node.Content[i+1]
			}
		}
		return nil
	}

	filterSequence := func(seq *yaml.Node, match func(*yaml.Node) bool) {
		if seq == nil || seq.Kind != yaml.SequenceNode {
			return
		}
		var newContent []*yaml.Node
		for _, item := range seq.Content {
			if !match(item) {
				newContent = append(newContent, item)
			}
		}
		seq.Content = newContent
	}

	// target values that should match
	targetStr := req.URL
	targetHost := host

	matchesTarget := func(val string) bool {
		v := strings.ToLower(val)
		ts := strings.ToLower(targetStr)
		th := strings.ToLower(targetHost)

		if v == ts || v == th || strings.Contains(v, "://"+ts) || strings.Contains(v, "://"+th) {
			return true
		}
		// Broadest fallback
		if strings.Contains(v, ts) || strings.Contains(v, th) {
			return true
		}
		return false
	}

	for _, doc := range m.Content {
		// Health & Latency & SSL & TCP
		for _, section := range []string{"health", "latency", "ssl_tls", "tcp_metrics"} {
			if secNode := findKey(doc, section); secNode != nil {
				if targetsNode := findKey(secNode, "targets"); targetsNode != nil {
					filterSequence(targetsNode, func(n *yaml.Node) bool {
						return n.Kind == yaml.ScalarNode && matchesTarget(n.Value)
					})
				}
			}
		}

		// Http Advanced
		if httpAdvNode := findKey(doc, "http_advanced"); httpAdvNode != nil {
			if targetsNode := findKey(httpAdvNode, "targets"); targetsNode != nil {
				filterSequence(targetsNode, func(n *yaml.Node) bool {
					if n.Kind == yaml.MappingNode {
						urlNode := findKey(n, "url")
						if urlNode != nil && matchesTarget(urlNode.Value) {
							return true
						}
					}
					// Also check if it's a simple scalar in http_advanced
					if n.Kind == yaml.ScalarNode && matchesTarget(n.Value) {
						return true
					}
					return false
				})
			}
		}

		// DNS Advanced
		if dnsNode := findKey(doc, "dns_advanced"); dnsNode != nil {
			if queriesNode := findKey(dnsNode, "queries"); queriesNode != nil {
				filterSequence(queriesNode, func(n *yaml.Node) bool {
					if n.Kind == yaml.MappingNode {
						nameNode := findKey(n, "name")
						if nameNode != nil && matchesTarget(nameNode.Value) {
							return true
						}
					}
					return false
				})
			}
		}

		// Traceroute
		if traceNode := findKey(doc, "traceroute"); traceNode != nil {
			if targetsNode := findKey(traceNode, "targets"); targetsNode != nil {
				filterSequence(targetsNode, func(n *yaml.Node) bool {
					if n.Kind == yaml.MappingNode {
						addrNode := findKey(n, "addr")
						if addrNode != nil && matchesTarget(addrNode.Value) {
							return true
						}
					}
					return false
				})
			}
		}
	}

	out, err := yaml.Marshal(&m)
	if err != nil {
		log.Error("Failed to marshal config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		log.Error("Failed to write config file", "error", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Trigger immediate reload
	var newCfg runtime.Config
	if err := yaml.Unmarshal(out, &newCfg); err == nil {
		s.cRuntime <- newCfg
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
