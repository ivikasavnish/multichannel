package http

import (
	"encoding/json"
	"fmt"
	"multichannel/screenshot"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

type ScreenshotHandler struct {
	manager *screenshot.ScreenshotManager
	mu      sync.Mutex
}

func NewScreenshotHandler() *ScreenshotHandler {
	return &ScreenshotHandler{
		manager: screenshot.NewScreenshotManager(),
	}
}

func (h *ScreenshotHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/screenshot/capture", h.CaptureMetrics).Methods(http.MethodPost)
	r.HandleFunc("/api/screenshot/progress", h.StreamProgress).Methods(http.MethodGet)
}

func (h *ScreenshotHandler) StreamProgress(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Listen for progress updates
	for progress := range h.manager.ProgressChan {
		data, err := json.Marshal(progress)
		if err != nil {
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (h *ScreenshotHandler) CaptureMetrics(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var req screenshot.CaptureOptions

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to parse request: "+err.Error(), http.StatusBadRequest)
		return
	}

	metrics, err := h.manager.CaptureMetrics(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure screenshot directory exists
	screenshotDir := "screenshots"
	if _, err := os.Stat(screenshotDir); os.IsNotExist(err) {
		os.Mkdir(screenshotDir, 0755)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"screenshot":      metrics.Screenshot,
		"title":           metrics.Title,
		"network_logs":    metrics.NetworkLogs,
		"console_logs":    metrics.ConsoleLogs,
		"device_settings": metrics.DeviceSettings,
	})
}
