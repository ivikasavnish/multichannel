package callbacks

import (
	"encoding/json"
	"log"
	"multichannel/cmd/typedefs"
	"multichannel/screenshot"
)

var screenshotManager *screenshot.ScreenshotManager

func init() {
	screenshotManager = screenshot.NewScreenshotManager()
}

// ScreenshotCallback handles screenshot requests
func ScreenshotCallback(req typedefs.Request) interface{} {
	// Parse request body if any parameters are needed
	var params map[string]interface{}
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &params); err != nil {
			log.Printf("Error parsing screenshot parameters: %v", err)
		}
	}

	// Capture screenshot
	base64Image, err := screenshotManager.CaptureScreen()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	// Return response
	return map[string]interface{}{
		"success":     true,
		"image":       base64Image,
		"timestamp":   screenshotManager.GetLastCaptureTime(),
		"image_type": "png",
		"encoding":   "base64",
	}
}
