<h1 >
<h1 align="center">
  <br>
  <a href="https://github.com/zaporter/viam-timelapse"><img src="https://raw.githubusercontent.com/zaporter/viam-timelapse/main/etc/icon.jpg" alt="stars long exposure icon" width="300"></a>
  <br>
  Timelapse module for Viam
  <br>
</h1>

# Features

# Models

```
zaporter:timelapse:v1
zaporter:timelapse:v1-fake
```

# Example Config

```json
{
  "capture_camera": "webcam_name_here",
  "capture_interval_seconds": "10",
  "playback_fps": "20",
}
```

# Output

A camera stream on the control page that shows the timelapse (if using the fake model, it will show a sample timelapse)

# Building

`make build` -> `bin/module`

# Linting

`make lint`
