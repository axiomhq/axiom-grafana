package plugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type aplWideFrameBuilder struct{}

func (aplWideFrameBuilder) Build(frame *data.Frame) (*data.Frame, error) {
	wideFrame, err := data.LongToWide(frame, &data.FillMissing{
		Mode: data.FillModePrevious,
	})
	if err != nil {
		return nil, err
	}

	applyWideFieldConfigs(wideFrame, frame)
	applyLabelDisplayNames(wideFrame)
	return wideFrame, nil
}

func applyWideFieldConfigs(wideFrame, longFrame *data.Frame) {
	configsByName := make(map[string]*data.FieldConfig, len(longFrame.Fields))
	for _, field := range longFrame.Fields {
		if field.Config != nil {
			configsByName[field.Name] = field.Config
		}
	}

	for _, field := range wideFrame.Fields {
		config := configsByName[field.Name]
		if config == nil {
			continue
		}
		field.Config = cloneFieldConfig(config)
	}
}

func cloneFieldConfig(config *data.FieldConfig) *data.FieldConfig {
	if config == nil {
		return nil
	}

	clone := *config
	if config.Custom != nil {
		clone.Custom = make(map[string]interface{}, len(config.Custom))
		for key, value := range config.Custom {
			clone.Custom[key] = value
		}
	}
	return &clone
}

func applyLabelDisplayNames(frame *data.Frame) {
	for _, field := range frame.Fields {
		if len(field.Labels) == 0 {
			continue
		}

		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.DisplayNameFromDS = labelsDisplayName(field.Labels)
	}
}

func labelsDisplayName(labels data.Labels) string {
	if len(labels) == 1 {
		for _, value := range labels {
			return value
		}
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, labels[key]))
	}

	return strings.Join(parts, ", ")
}
