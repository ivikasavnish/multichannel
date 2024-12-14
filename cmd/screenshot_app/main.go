package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	httpStd "net/http"
	"os"
	"path/filepath"
	"time"

	"multichannel/http"

	"github.com/gorilla/mux"
)

type ScreenshotRequest struct {
	URL            string  `json:"url"`
	DeviceType     string  `json:"device_type"`
	CaptureNetwork bool    `json:"capture_network"`
	CaptureConsole bool    `json:"capture_console"`
	Width          int64   `json:"width"`
	Height         int64   `json:"height"`
	Scale          float64 `json:"scale"`
	HeadlessMode   bool    `json:"headless_mode"`
}

type ScreenshotResponse struct {
	Success        bool                   `json:"success"`
	Image          string                 `json:"image,omitempty"`
	Title          string                 `json:"title,omitempty"`
	NetworkLogs    []string               `json:"network_logs,omitempty"`
	ConsoleLogs    []string               `json:"console_logs,omitempty"`
	DeviceSettings map[string]interface{} `json:"device_settings,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// Package level declaration
var screenshotHandler *http.ScreenshotHandler

func main() {
	// Initialize screenshot handler
	screenshotHandler = http.NewScreenshotHandler()

	// Create router
	r := mux.NewRouter()

	// Register screenshot routes
	screenshotHandler.RegisterRoutes(r)

	// Serve static files from the correct directory
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "web/static")
	r.PathPrefix("/static/").Handler(httpStd.StripPrefix("/static/", httpStd.FileServer(httpStd.Dir(staticDir))))

	// Serve the main page
	r.HandleFunc("/", func(w httpStd.ResponseWriter, r *httpStd.Request) {
		httpStd.ServeFile(w, r, filepath.Join(staticDir, "screenshot.html"))
	}).Methods("GET")

	// Start server
	port := 8090
	fmt.Printf("Starting screenshot server on http://localhost:%d\n", port)
	log.Fatal(httpStd.ListenAndServe(fmt.Sprintf(":%d", port), r))
}

func homeHandler(w httpStd.ResponseWriter, r *httpStd.Request) {
	tmplPath := filepath.Join("web", "static", "screenshot.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		httpStd.Error(w, "Failed to load template", httpStd.StatusInternalServerError)
		return
	}

	data := struct {
		Title string
		Port  int
	}{
		Title: "Screenshot Taker",
		Port:  8090,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		httpStd.Error(w, "Failed to render template", httpStd.StatusInternalServerError)
	}
}

func takeScreenshotHandler(w httpStd.ResponseWriter, r *httpStd.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(httpStd.StatusOK)
		return
	}

	// Parse request body
	var req ScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ScreenshotResponse{
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse request: %v", err),
			Timestamp: time.Now(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate URL
	if req.URL == "" {
		req.URL = "about:blank"
	}

	// Set default device type if not specified
	if req.DeviceType == "" {
		req.DeviceType = "desktop"
	}

	// Take screenshot with options
	screenshotHandler.CaptureMetrics(w, r)
	return
}

func healthCheckHandler(w httpStd.ResponseWriter, r *httpStd.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
