<h1 >
<h1 align="center">
  <br>
  <a href="https://github.com/zaporter/viam-timelapse"><img src="https://raw.githubusercontent.com/zaporter/viam-timelapse/main/etc/icon.jpg" alt="stars long exposure icon" width="300"></a>
  <br>
  Timelapse module for Viam
  <br>
</h1>

<h2>Warning: this module is in beta its config may change in future versions.</h2>

Captures frames from a camera at a designated frequency, saves them to the local device in ($VIAM\_MODULE\_DATA) and then plays back the timelapse on the control tab

# Models

```
zaporter:timelapse:v1
zaporter:timelapse:v1-fake
```

# Example Config

```json
{
  "capture_camera": "webcam_name_here",
  "capture_interval_seconds": 10.0,
  "playback_fps": 20,
  "timelapse_name" : "default" #optional
}
```

# Output

A camera stream on the control page that shows the timelapse (if using the fake model, it will show a sample timelapse)

# Missing functionality

Save or export the timelapse: I want to do this eventually. If someone wants this, feel free to file an issue and I'll find time to do it. Otherwise, you can just grab the images yourself and use imagemagick

Intelligently pick frames for the timelapse: Right now, the module just shows all frames. However, this is likely not what you want for long timelapses because it would show things like nighttime. It would be best to pick clear frames (low noise) that have a high similarity to either the previous frame or to a reference frame (perhaps the average of the last n frames?)

Preserve timelapse frame time between viam-server restarts: If you are set to take a photo 1x per day and restart the module at midnight, the clock will shift and it will start taking photos at midnight instead of at the middle of the day. There should be some way to say when to take photos (maybe cronjob syntax?)

# Building

`make build` -> `bin/module`

# Linting

`make lint`
