// Package sds011 is the package for sds011
package sds011

import (
	"context"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/gostream"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/pointcloud"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/rimage/transform"
)

var (
	Model     = resource.NewModel("zaporter", "timelapse", "v1")
	ModelFake = resource.NewModel("zaporter", "timelapse", "v1-fake")
)

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
	return instance, nil
}

// DoCommand sends/receives arbitrary data.
func (c *component) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

// Images is used for getting simultaneous images from different imagers,
// along with associated metadata (just timestamp for now). It's not for getting a time series of images from the same imager.
func (c *component) Images(ctx context.Context) ([]camera.NamedImage, resource.ResponseMetadata, error) {
	if c.isFake {
		return nil, resource.ResponseMetadata{}, nil
	}
	return c.captureCam.Images(ctx)
}

// Stream returns a stream that makes a best effort to return consecutive images
// that may have a MIME type hint dictated in the context via gostream.WithMIMETypeHint.
func (c *component) Stream(ctx context.Context, errHandlers ...gostream.ErrorHandler) (gostream.VideoStream, error) {
	if c.isFake {
		return nil, nil
	}
	return c.captureCam.Stream(ctx, errHandlers...)
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
		return camera.Properties{}, nil
	}
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
