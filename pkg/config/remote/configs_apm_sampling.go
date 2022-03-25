package remote

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

// APMSamplingConfig is an apm sampling config
type APMSamplingConfig struct {
	Versions map[string]uint64
	Files    map[string]pb.APMSampling
}

// APMSamplingUpdate is an apm sampling config update
type APMSamplingUpdate struct {
	Config *APMSamplingConfig
}

type apmSamplingConfigs struct {
	config *APMSamplingConfig
}

func newApmSamplingConfigs() *apmSamplingConfigs {
	return &apmSamplingConfigs{}
}

func (c *apmSamplingConfigs) update(configFiles map[string]configFiles) (*APMSamplingUpdate, error) {
	if !c.shouldUpdate(configFiles) {
		return nil, nil
	}
	update := &APMSamplingUpdate{
		Config: &APMSamplingConfig{
			Versions: make(map[string]uint64, len(configFiles)),
			Files:    make(map[string]pb.APMSampling, len(configFiles)),
		},
	}
	for _, files := range configFiles {
		for _, file := range files {
			var mpconfig pb.APMSampling
			_, err := mpconfig.UnmarshalMsg(file.raw)
			if err != nil {
				return nil, fmt.Errorf("could not parse apm sampling config: %v", err)
			}
			update.Config.Versions[file.pathMeta.ConfigID] = file.version
			update.Config.Files[file.pathMeta.ConfigID] = mpconfig
		}
		c.config = update.Config
	}
	return update, nil
}

func (c *apmSamplingConfigs) shouldUpdate(configFiles map[string]configFiles) bool {
	if c.config == nil || len(c.config.Versions) != len(configFiles) {
		return true
	}
	for configID, files := range configFiles {
		if version, ok := c.config.Versions[configID]; !ok || version < files.version() {
			return true
		}
	}
	return false
}
