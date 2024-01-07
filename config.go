package sds011

import (
	"errors"
)

type Config struct {
	USBInterface string `json:"usb_interface"`
}

// Validate takes the current location in the config (useful for good error messages).
// It should return a []string which contains all of the implicit
// dependencies of a module. (or nil,err if the config does not pass validation).
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.USBInterface == "" {
		return nil, errors.New(path + " usb_interface must be non-empty. Example value: /dev/ttyUSB0")
	}
	return make([]string, 0), nil
}
