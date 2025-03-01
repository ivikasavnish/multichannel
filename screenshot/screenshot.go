package screenshot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// DeviceType represents different device types for emulation
type DeviceType string

const (
	Desktop DeviceType = "desktop"
	Mobile  DeviceType = "mobile"
	Tablet  DeviceType = "tablet"
)

// CaptureOptions contains settings for screenshot capture
type CaptureOptions struct {
	URL            string     `json:"url"`
	DeviceType     DeviceType `json:"device_type"`
	CaptureNetwork bool       `json:"capture_network"`
	CaptureConsole bool       `json:"capture_console"`
	Width          int64      `json:"width"`
	Height         int64      `json:"height"`
	Scale          float64    `json:"scale"`
	HeadlessMode   bool       `json:"headless_mode"`
	FullPage       bool       `json:"full_page"` // New field for full page screenshot
}

// BrowserMetrics contains various metrics captured from the browser
type BrowserMetrics struct {
	Screenshot     []byte                 `json:"screenshot"`
	Title          string                 `json:"title"`
	NetworkLogs    []string               `json:"network_logs,omitempty"`
	ConsoleLogs    []string               `json:"console_logs,omitempty"`
	LastCapture    time.Time              `json:"last_capture"`
	DeviceSettings map[string]interface{} `json:"device_settings"`
}

// Progress represents a progress update during screenshot capture
type Progress struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// ScreenshotManager handles browser interactions and metrics collection
type ScreenshotManager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	ProgressChan chan Progress
}

// NewScreenshotManager creates a new screenshot manager with CDP support
func NewScreenshotManager() *ScreenshotManager {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)

	// Create base context with timeout
	baseCtx, baseCancel := context.WithTimeout(context.Background(), 2*time.Minute)

	allocCtx, _ := chromedp.NewExecAllocator(baseCtx, opts...)

	// Create browser context with debug logging
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)

	// Ensure browser is started
	if err := chromedp.Run(ctx); err != nil {
		log.Printf("Failed to start browser: %v", err)
		baseCancel()
		cancel()
		return nil
	}

	return &ScreenshotManager{
		ctx:         ctx,
		cancel:      cancel,
		ProgressChan: make(chan Progress, 10),
	}
}

// getDeviceSettings returns device-specific settings
func getDeviceSettings(deviceType DeviceType, width, height int64) map[string]interface{} {
	switch deviceType {
	case Mobile:
		if width == 0 {
			width = 375
		}
		if height == 0 {
			height = 812
		}
		return map[string]interface{}{
			"width":             width,
			"height":            height,
			"deviceScaleFactor": 2.0,
			"mobile":            true,
			"userAgent":         "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		}
	case Tablet:
		if width == 0 {
			width = 768
		}
		if height == 0 {
			height = 1024
		}
		return map[string]interface{}{
			"width":             width,
			"height":            height,
			"deviceScaleFactor": 2.0,
			"mobile":            true,
			"userAgent":         "Mozilla/5.0 (iPad; CPU OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
		}
	default: // Desktop
		if width == 0 {
			width = 1920
		}
		if height == 0 {
			height = 1080
		}
		return map[string]interface{}{
			"width":             width,
			"height":            height,
			"deviceScaleFactor": 1.0,
			"mobile":            false,
			"userAgent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		}
	}
}

// CaptureMetrics captures a screenshot and page metrics based on provided options
func (sm *ScreenshotManager) CaptureMetrics(opts CaptureOptions) (*BrowserMetrics, error) {
	// Create a timeout context for this capture - increased timeout
	ctx, cancel := context.WithTimeout(sm.ctx, 120*time.Second)
	defer cancel()

	// Create options for the browser
	browserOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", opts.HeadlessMode),
		// Show the browser UI in non-headless mode
		chromedp.Flag("start-maximized", !opts.HeadlessMode),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("hide-scrollbars", false),
		// Add additional flags for better performance
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// Create a new ExecAllocator
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, browserOpts...)
	defer cancel()

	// Create a new browser context
	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Initialize metrics
	metrics := &BrowserMetrics{
		LastCapture:    time.Now(),
		DeviceSettings: getDeviceSettings(opts.DeviceType, opts.Width, opts.Height),
	}

	// Configure browser options
	var screenshot []byte
	var title string
	var networkLogs, consoleLogs []string

	// Navigate to the page first
	sm.ProgressChan <- Progress{Stage: "navigation", Message: "Navigating to page..."}
	if err := chromedp.Run(taskCtx, chromedp.Navigate(opts.URL)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}

	// Set viewport after navigation if dimensions are provided
	if opts.Width > 0 && opts.Height > 0 {
		sm.ProgressChan <- Progress{Stage: "viewport", Message: "Setting viewport dimensions..."}
		if err := chromedp.Run(taskCtx, chromedp.EmulateViewport(opts.Width, opts.Height, chromedp.EmulateScale(opts.Scale))); err != nil {
			log.Printf("Warning: failed to set viewport: %v", err)
			// Continue execution even if viewport setting fails
		}
	}

	// Add network logging if requested
	if opts.CaptureNetwork {
		chromedp.ListenTarget(taskCtx, func(ev interface{}) {
			switch e := ev.(type) {
			case *network.EventRequestWillBeSent:
				networkLogs = append(networkLogs, fmt.Sprintf("Request: %s %s", e.Request.Method, e.Request.URL))
			case *network.EventResponseReceived:
				networkLogs = append(networkLogs, fmt.Sprintf("Response: %s %d [%s]", e.Response.URL, e.Response.Status, e.Response.StatusText))
			}
		})
	}

	// Add console logging if requested
	if opts.CaptureConsole {
		chromedp.ListenTarget(taskCtx, func(ev interface{}) {
			if e, ok := ev.(*runtime.EventConsoleAPICalled); ok {
				var args []string
				for _, arg := range e.Args {
					args = append(args, string(arg.Value))
				}
				consoleLogs = append(consoleLogs, fmt.Sprintf("[%s] %s", e.Type, strings.Join(args, " ")))
			}
		})
	}

	// Wait for the page to be ready with network idle
	sm.ProgressChan <- Progress{Stage: "loading", Message: "Waiting for page to load..."}
	
	// First wait for document to be ready
	if err := chromedp.Run(taskCtx, chromedp.Tasks{
		network.Enable(),
		chromedp.WaitReady("body", chromedp.ByQuery),
	}); err != nil {
		log.Printf("Warning: initial page load check failed: %v", err)
		// Continue anyway as some pages might not trigger ready state properly
	}

	// Then wait for network to be relatively idle
	if err := chromedp.Run(taskCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		sm.ProgressChan <- Progress{Stage: "network", Message: "Waiting for network to be idle..."}
		
		// Wait a shorter time for network idle
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
				// Continue waiting
			}
		}
		return nil
	})); err != nil {
		log.Printf("Warning: network idle check failed: %v", err)
		// Continue anyway as we might still get a good screenshot
	}

	sm.ProgressChan <- Progress{Stage: "capture", Message: "Capturing screenshot..."}
	// Capture screenshot and other metrics
	if opts.FullPage {
		if err := chromedp.Run(taskCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			// Get page height
			var height int64
			if err := chromedp.Evaluate(`
				Math.min(
					Math.max(
						document.documentElement.scrollHeight,
						document.body.scrollHeight
					),
					window.innerHeight * 50  // Maximum 50 viewport heights
				)
			`, &height).Do(ctx); err != nil {
				return err
			}

			// Take full page screenshot
			return chromedp.FullScreenshot(&screenshot, 100).Do(ctx)
		})); err != nil {
			return nil, fmt.Errorf("failed to capture full page screenshot: %w", err)
		}
	} else {
		if err := chromedp.Run(taskCtx, chromedp.Tasks{
			chromedp.Title(&title),
			chromedp.CaptureScreenshot(&screenshot),
		}); err != nil {
			return nil, fmt.Errorf("failed to capture page content: %w", err)
		}
	}

	metrics.Screenshot = screenshot
	metrics.Title = title
	metrics.NetworkLogs = networkLogs
	metrics.ConsoleLogs = consoleLogs

	return metrics, nil
}

// Close cleans up resources
func (sm *ScreenshotManager) Close() {
	if sm.cancel != nil {
		sm.cancel()
	}
}
