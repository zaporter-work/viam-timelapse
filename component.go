// Package sds011 is the package for sds011
package sds011

import (
	"bytes"
	"context"
	"embed"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
	"go.viam.com/utils"
)

var (
	Model     = resource.NewModel("zaporter", "timelapse", "v1")
	ModelFake = resource.NewModel("zaporter", "timelapse", "v1-fake")
	DataDir   = os.Getenv("VIAM_MODULE_DATA")

	FakeFolder = filepath.Join(DataDir, "fake")
)

//go:embed etc/fake_timelapse
var fakeImages embed.FS

func init() {
	registration := resource.Registration[resource.Resource, *Config]{
		Constructor: func(ctx context.Context,
			deps resource.Dependencies,
			conf resource.Config,
			logger logging.Logger,
		) (resource.Resource, error) {
			return createComponent(ctx, deps, conf, logger, false)
		},
	}
	resource.RegisterComponent(camera.API, Model, registration)

	registrationFake := resource.Registration[resource.Resource, *Config]{
		Constructor: func(ctx context.Context,
			deps resource.Dependencies,
			conf resource.Config,
			logger logging.Logger,
		) (resource.Resource, error) {
			return createComponent(ctx, deps, conf, logger, true)
		},
	}
	resource.RegisterComponent(camera.API, ModelFake, registrationFake)
}

type component struct {
	resource.Named
	resource.AlwaysRebuild
	cfg    *Config
	isFake bool

	stream     *TimelapseStream
	captureCam camera.Camera

	cancelCtx  context.Context
	cancelFunc func()

	logger logging.Logger
}

func createComponent(_ context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
	isFake bool,
) (camera.Camera, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, errors.Wrap(err, "create component failed due to config parsing")
	}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	captureCam := camera.Camera(nil)
	if !isFake {
		captureCam, err = camera.FromDependencies(deps, newConf.CaptureCamera)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create timelapse because we could not access the camera")
		}
	} else {
		if err := createFakeFiles(); err != nil {
			return nil, err
		}
		newConf.TimelapseName = "fake"
	}

	instance := &component{
		Named:      conf.ResourceName().AsNamed(),
		cfg:        newConf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		captureCam: captureCam,
		isFake:     isFake,
		logger:     logger,
	}
	if !instance.isFake {
		if err := os.MkdirAll(filepath.Join(DataDir, instance.cfg.TimelapseName), 0o700); err != nil {
			return nil, err
		}
		instance.startBgProcess()
	}
	return instance, nil
}

func (c *component) startBgProcess() {
	utils.PanicCapturingGo(func() {
		ticker := time.NewTicker(time.Duration(c.cfg.CaptureIntervalSeconds * float64(time.Second)))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.logger.Info("capturing img")
				imgs, _, err := c.captureCam.Images(c.cancelCtx)
				if err != nil {
					c.logger.Errorw("cannot read img", "err", err)
					continue
				}
				for _, img := range imgs {
					buf := new(bytes.Buffer)
					err := jpeg.Encode(buf, img.Image, nil)
					if err != nil {
						c.logger.Errorw("cannot encode img as jpeg", "err", err)
						continue
					}
					imgBytes := buf.Bytes()
					err = os.WriteFile(filepath.Join(DataDir, c.cfg.TimelapseName, time.Now().Format(time.RFC3339Nano)+".jpg"), imgBytes, 0o777)
					if err != nil {
						c.logger.Errorw("cannot write img", "err", err)
						continue
					}

				}

			case <-c.cancelCtx.Done():
				c.logger.Info("shutdown")
				return
			}
		}
	})
}

func createFakeFiles() error {
	if err := os.MkdirAll(FakeFolder, 0o700); err != nil {
		return err
	}

	writeFake := func(name string) error {
		data, err := fakeImages.ReadFile(name)
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(FakeFolder, filepath.Base(name)), data, 0o777)
		return err
	}
	if err := writeFake("etc/fake_timelapse/1_1.jpg"); err != nil {
		return err
	}
	if err := writeFake("etc/fake_timelapse/1_2.jpg"); err != nil {
		return err
	}
	return nil
}

// DoCommand sends/receives arbitrary data.
func (c *component) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
    return map[string]interface{}{"unimplemented":"reach out to me with what you want"}, nil
}

// Images is used for getting simultaneous images from different imagers,
// along with associated metadata (just timestamp for now). It's not for getting a time series of images from the same imager.
func (c *component) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	if c.isFake {
		images := []camera.NamedImage{
			{SourceName: "zack", Image: image.NewRGBA(image.Rect(0, 0, 100, 100))},
		}
		return images, resource.ResponseMetadata{CapturedAt: time.Now()}, nil
	}
	return c.captureCam.Images(ctx)
}

// Stream returns a stream that makes a best effort to return consecutive images
// that may have a MIME type hint dictated in the context via gostream.WithMIMETypeHint.
func (c *component) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	if c.stream == nil {
		c.stream = &TimelapseStream{
			imgDir:         filepath.Join(DataDir, c.cfg.TimelapseName),
			logger:         c.logger,
			last_swap_time: time.Now(),
			swap_interval:  time.Duration(float64(time.Second) * 1 / c.cfg.PlaybackFPS),
		}
	}
	return c.stream, nil
}

type TimelapseStream struct {
	imgDir         string
	logger         logging.Logger
	last_swap_time time.Time
	swap_interval  time.Duration
	index          int
}

func (t *TimelapseStream) NextPath() (string, error) {
	dir, err := os.Open(t.imgDir)
	if err != nil {
		return "", err
	}
	entries, err := dir.Readdirnames(0)
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	if time.Since(t.last_swap_time) >= t.swap_interval {
		t.index += 1
		t.last_swap_time = time.Now()
	}
	if t.index >= len(entries) {
		t.index = 0
	}
	return filepath.Join(t.imgDir, entries[t.index]), nil
}
func readImg(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	return img, err
}

// Next returns the next media element in the sequence (best effort).
// Note: This element is mutable and shared globally; it MUST be copied
// before it is mutated.
func (t *TimelapseStream) Next(ctx context.Context) (image.Image, func(), error) {
	diff := time.Since(t.last_swap_time)
	if diff <= t.swap_interval {
		time.Sleep(diff - t.swap_interval)
	}
	path, err := t.NextPath()
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, 100, 100)), func() {}, err
	}
	img, err := readImg(path)
	return img, func() {}, err
}

// Close signals this stream is no longer needed and releases associated
// resources.
func (t *TimelapseStream) Close(ctx context.Context) error {
	return nil
}

// NextPointCloud returns the next immediately available point cloud, not necessarily one
// a part of a sequence. In the future, there could be streaming of point clouds.
func (c *component) NextPointCloud(ctx context.Context) (pointcloud.PointCloud, error) {
	if c.isFake {
		return nil, nil
	}
	return c.captureCam.NextPointCloud(ctx)
}

// Properties returns properties that are intrinsic to the particular
// implementation of a camera
func (c *component) Properties(ctx context.Context) (camera.Properties, error) {
	if c.isFake {
		return camera.Properties{
			SupportsPCD:      false,
			ImageType:        camera.ImageType("color"),
			IntrinsicParams:  &transform.PinholeCameraIntrinsics{},
			DistortionParams: nil,
			MimeTypes:        []string{"image/jpeg", "image/png"},
		}, nil
	}
	c.logger.Debug(c.captureCam.Properties(ctx))
	return c.captureCam.Properties(ctx)
}

func (c *component) Projector(ctx context.Context) (transform.Projector, error) {
	if c.isFake {
		return nil, nil
	}
	return c.captureCam.Projector(ctx)
}

// Close must safely shut down the resource and prevent further use.
// Close must be idempotent.
// Later reconfiguration may allow a resource to be "open" again.
func (c *component) Close(ctx context.Context) error {
	c.cancelFunc()
	c.logger.Info("closing\n")
	return nil
}
