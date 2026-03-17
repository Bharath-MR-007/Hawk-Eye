package hawkeye

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bharath-MR-007/hawk-eye/internal/logger"
)

type ConfigExportRequest struct {
	Tags string `json:"tags"`
}

type ConfigImportResponse struct {
	Tags string `json:"tags"`
}

func (s *Hawkeye) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Info("Handling configuration export request")

	var req ConfigExportRequest
	if r.Method == http.MethodPost && r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Warn("Could not decode export body", "error", err)
		}
	}

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	count := 0
	// 1. Files to include
	configs := []string{
		"checks.yaml",
		"prometheus_rules.yaml",
		"prometheus.yaml",
		"docker-compose.yaml",
		"Dockerfile",
		"live_dashboard.html",
		"login.html",
		"inventory.html",
		"alerts_config.html",
		"polling_config.html",
	}

	for _, f := range configs {
		if err := findAndAddFileToZip(zw, f); err == nil {
			count++
		} else {
			log.Debug("Skipping file during export", "file", f, "error", err)
		}
	}

	// 2. Essential Directories
	dirs := []string{
		"alertmanager",
		"grafana",
		"chart",
		"scripts",
	}

	for _, d := range dirs {
		if err := findAndAddDirToZip(zw, d); err == nil {
			count++
		} else {
			log.Debug("Skipping directory during export", "dir", d, "error", err)
		}
	}

	// 3. Browser Tags
	if req.Tags != "" {
		tf, err := zw.Create("tags.json")
		if err == nil {
			tf.Write([]byte(req.Tags))
			count++
		}
	}

	// Add a small manifest
	mf, err := zw.Create("export-manifest.json")
	if err == nil {
		manifest := map[string]string{
			"version":   "1.0.0",
			"exported":  time.Now().Format(time.RFC3339),
			"source":    "Hawkeye Full Instance Export",
			"fileCount": fmt.Sprintf("%d", count),
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		mf.Write(data)
	}

	if err := zw.Close(); err != nil {
		log.Error("Failed to finalize zip", "error", err)
		http.Error(w, "Failed to create archive", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		log.Error("No configuration files were found to export")
		http.Error(w, "No configuration files found in / or current directory", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=hawk-eye-full-config.zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())

	log.Info("Successfully exported configuration", "size", buf.Len(), "files", count)
}

func findAndAddFileToZip(zw *zip.Writer, filename string) error {
	// Try absolute root first (for Docker env)
	path := filepath.Join("/", filename)
	if _, err := os.Stat(path); err == nil {
		return addFileToZip(zw, path, filename)
	}
	// Try current directory
	if _, err := os.Stat(filename); err == nil {
		return addFileToZip(zw, filename, filename)
	}
	return os.ErrNotExist
}

func findAndAddDirToZip(zw *zip.Writer, dirName string) error {
	path := filepath.Join("/", dirName)
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return addDirToZip(zw, path, dirName)
	}
	if st, err := os.Stat(dirName); err == nil && st.IsDir() {
		return addDirToZip(zw, dirName, dirName)
	}
	return os.ErrNotExist
}

func (s *Hawkeye) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	file, _, err := r.FormFile("config")
	if err != nil {
		log.Error("Failed to get config file from upload", "error", err)
		http.Error(w, "Failed to get config file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpFile, err := os.CreateTemp("", "hawk-eye-config-*.zip")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		http.Error(w, "Failed to save temp file", http.StatusInternalServerError)
		return
	}

	zr, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		http.Error(w, "Failed to open zip reader", http.StatusBadRequest)
		return
	}
	defer zr.Close()

	var tags string
	for _, f := range zr.File {
		if f.Name == "tags.json" {
			rc, err := f.Open()
			if err == nil {
				content, _ := io.ReadAll(rc)
				tags = string(content)
				rc.Close()
			}
			continue
		}

		if strings.Contains(f.Name, "..") {
			continue
		}

		// Determine destination path
		// We try to write back to where the application expects them
		// In Docker, we might need to write to / if that's where they are
		destPath := f.Name

		// If we are running in an environment where files are usually in /,
		// but the zip has them as "checks.yaml", we might want to be careful.
		// However, most deployments use the current working directory.

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			log.Error("Failed to create dir", "dir", filepath.Dir(destPath), "error", err)
			continue
		}

		dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Error("Failed to open dest file", "file", destPath, "error", err)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			dstFile.Close()
			continue
		}

		_, err = io.Copy(dstFile, rc)
		dstFile.Close()
		rc.Close()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConfigImportResponse{Tags: tags})

	// Trigger reloads in background
	go func() {
		// Reload Prometheus
		http.Post("http://prometheus:9090/-/reload", "", nil)
		// Reload Alertmanager
		http.Post("http://alertmanager:9093/-/reload", "", nil)
	}()
}

func addFileToZip(zw *zip.Writer, realPath, zipPath string) error {
	info, err := os.Stat(realPath)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(realPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}

func addDirToZip(zw *zip.Writer, rootPath, zipPrefix string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, path)
		return addFileToZip(zw, path, filepath.Join(zipPrefix, relPath))
	})
}
