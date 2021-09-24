// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package generic

import (
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
)

// MetricsAdapter provides a way to change metrics and tags before sending them out
type MetricsAdapter interface {
	AdaptTags(tags []string, c workloadmeta.Container) []string
	AdaptMetrics(metricName string, value float64) (string, float64)
}

type ContainerLister interface {
	List() ([]workloadmeta.Container, error)
}

type MetadataContainerLister struct{}

func (l MetadataContainerLister) List() ([]workloadmeta.Container, error) {
	return workloadmeta.GetGlobalStore().ListContainers()
}

type GenericMetricsAdapter struct{}

func (a GenericMetricsAdapter) AdaptTags(tags []string, c workloadmeta.Container) []string {
	return append(tags, "runtime:"+string(c.Runtime))
}

func (a GenericMetricsAdapter) AdaptMetrics(metricName string, value float64) (string, float64) {
	return metricName, value
}
