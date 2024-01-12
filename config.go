package sds011

import (
	"errors"
)

type Config struct {
	CaptureCamera          string  `json:"capture_camera"`
	CaptureIntervalSeconds float64 `json:"capture_interval_seconds"`
	PlaybackFPS            float64 `json:"playback_fps"`
	TimelapseName          string  `json:"timelapse_name"`
}

// Validate takes the current location in the config (useful for good error messages).
// It should return a []string which contains all of the implicit
// dependencies of a module. (or nil,err if the config does not pass validation).
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.CaptureCamera == "" {
		return nil, errors.New(path + " capture_camera must be non-empty. Example value: your_webcam_name_here")
	}
	if cfg.CaptureIntervalSeconds <= 0 {
		return nil, errors.New(path + " capture_interval_seconds must be greater than 0")
	}
	if cfg.PlaybackFPS <= 0 {
		return nil, errors.New(path + " playback_fps must be greater than 0")
	}
    if cfg.TimelapseName == "" {
        cfg.TimelapseName = "default"
    }
	return []string{cfg.CaptureCamera}, nil
}
