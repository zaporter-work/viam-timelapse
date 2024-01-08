// Package sds011 is the package for sds011
package sds011

import (
	"context"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
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
	cfg          *Config
	isFake       bool
	sds011Sensor *sds011.Sensor

	cancelCtx  context.Context
	cancelFunc func()

	logger logging.Logger
}

func createComponent(_ context.Context,
	_ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
	isFake bool,
) (sensor.Sensor, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, errors.Wrap(err, "create component failed due to config parsing")
	}

	var sensor *sds011.Sensor
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	instance := &component{
		Named:        conf.ResourceName().AsNamed(),
		cfg:          newConf,
		cancelCtx:    cancelCtx,
		cancelFunc:   cancelFunc,
		sds011Sensor: sensor,
		isFake:       isFake,
		logger:       logger,
	}
	if !isFake {
		if err := instance.setupSensor(); err != nil {
			if instance.sds011Sensor != nil {
				instance.sds011Sensor.Close()
			}
			return nil, err
		}
	}
	return instance, nil
}

func (c *component) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	if c.isFake {
		return map[string]interface{}{
			"pm_10":  10.0,
			"pm_2.5": 15.0,
			"units":  "μg/m³",
		}, nil
	}
	reading, err := c.sds011Sensor.Query()
	if err != nil {
		// try resetting the sensor
		if err2 := c.setupSensor(); err2 != nil {
			return nil, errors.Wrap(err, err2.Error())
		}
		reading, err = c.sds011Sensor.Query()
		if err != nil {
			return nil, err
		}
	}
	return map[string]interface{}{
		"pm_10":  reading.PM10,
		"pm_2.5": reading.PM25,
		"units":  "μg/m³",
	}, nil
}

func (c *component) setupSensor() error {
	c.logger.Info("setting up sensor\n")
	if c.sds011Sensor != nil {
		c.sds011Sensor.Close()
	}
	var err error
	c.sds011Sensor, err = sds011.New(c.cfg.USBInterface)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to interface %q", c.cfg.USBInterface)
	}
	if val, err := c.sds011Sensor.IsAwake(); err != nil || !val {
		if err != nil {
			return errors.Wrap(err, "reading sensor awakeness")
		}
		if err := c.sds011Sensor.Awake(); err != nil {
			return errors.Wrap(err, "unable to set the sensor to awake")
		}
	}
	if err := c.sds011Sensor.MakePassive(); err != nil {
		return errors.Wrap(err, "unable to set the sensor to passive")
	}
	return nil
}

// DoCommand sends/receives arbitrary data.
func (c *component) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

// Close must safely shut down the resource and prevent further use.
// Close must be idempotent.
// Later reconfiguration may allow a resource to be "open" again.
func (c *component) Close(ctx context.Context) error {
	c.cancelFunc()
	if c.sds011Sensor != nil {
		c.sds011Sensor.Close()
	}
	c.logger.Info("closing\n")
	return nil
}
