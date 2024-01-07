
<h1 >
<h1 align="center">
  <br>
  <a href="https://github.com/zaporter-work/viam-sds011"><img src="https://raw.githubusercontent.com/zaporter-work/viam-sds011/main/etc/sds011.jpg" alt="SDS011 image" width="200"></a>
  <br>
  SDS011 Air quality sensor module for Viam
  <br>
</h1>

# Features
- hot reloading
- basic functionality

# Models
```
zaporter:sds011:v1
zaporter:sds011:v1-fake
```
# Example Config
```json
{
  "usb_interface": "/dev/serial/by-id/usb-1a86_USB_Serial-if00-port0"
}
```
# Output
```json5
{
  "pm_10": float64, 
  "pm_2.5": float64,
  "units": "μg/m³"
}
```

# Building
`make build` -> `bin/module`

# Linting
`make lint`
