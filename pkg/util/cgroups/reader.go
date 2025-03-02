// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Reader is the main interface to scrape data from cgroups
type Reader struct {
	hostPrefix             string
	procPath               string
	cgroupVersion          int
	cgroupV1BaseController string
	readerFilter           ReaderFilter
	impl                   readerImpl

	cgroups         map[string]Cgroup
	scrapeTimestmap time.Time
	cacheValidFor   time.Duration
}

type readerImpl interface {
	parseCgroups() (map[string]Cgroup, error)
}

// ReaderFilter allows to filter cgroups based on their path + folder name
type ReaderFilter func(path, name string) (string, error)

// DefaultFilter matches all cgroup folders and use folder name as identifier
func DefaultFilter(path, name string) (string, error) {
	return path, nil
}

var containerRegexp = regexp.MustCompile("([0-9a-f]{64}|[0-9a-f]{8}(-[0-9a-f]{4}){4})")

// ContainerFilter returns a filter that will match cgroup folders containing a container id
func ContainerFilter(path, name string) (string, error) {
	match := containerRegexp.FindString(name)

	// With systemd cgroup driver, there may be a `.mount` cgroup on top of the normal one
	// While existing, no process is attached to it and thus holds no stats
	if match != "" {
		if strings.HasSuffix(name, ".mount") {
			return "", nil
		}

		return match, nil
	}

	return "", nil
}

// ReaderOption allows to customize reader behavior (Builder-style)
type ReaderOption func(*Reader)

// WithHostPrefix sets where hosts path are mounted (if not running on-host)
func WithHostPrefix(hostPrefix string) ReaderOption {
	return func(r *Reader) {
		r.hostPrefix = hostPrefix
	}
}

// WithProcPath sets where /proc is currently mounted.
// If set, hostPrefix is not added to this path.
// Default to `$hostPrefix/proc` if empty.
func WithProcPath(fullPath string) ReaderOption {
	return func(r *Reader) {
		r.procPath = fullPath
	}
}

// WithReaderFilter sets the filter used to select interesting cgroup folders
// and provides an identifier for them.
func WithReaderFilter(rf ReaderFilter) ReaderOption {
	return func(r *Reader) {
		r.readerFilter = rf
	}
}

// WithCgroupV1BaseController sets which controller is used to select cgroups
// it then assumes that, if being, used other controllers uses the same relative path.
// Default to "memory" if not set.
func WithCgroupV1BaseController(controller string) ReaderOption {
	return func(r *Reader) {
		r.cgroupV1BaseController = controller
	}
}

// WithCgroupPathsCache sets the duration during which the discovered cgroup paths are considered valid.
// Note that only the paths are cached, not the statistics.
// Default to 0 if not set (no cache).
func WithCgroupPathsCache(validFor time.Duration) ReaderOption {
	return func(r *Reader) {
		r.cacheValidFor = validFor
	}
}

// NewReader returns a new cgroup reader with given options
func NewReader(opts ...ReaderOption) (*Reader, error) {
	r := &Reader{}
	for _, opt := range opts {
		opt(r)
	}

	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reader) init() error {
	if r.procPath == "" {
		r.procPath = filepath.Join(r.hostPrefix, "/proc")
	}

	cgroupMounts, err := discoverCgroupMountPoints(r.hostPrefix, r.procPath)
	if err != nil {
		return err
	}

	if r.readerFilter == nil {
		r.readerFilter = DefaultFilter
	}

	if isCgroup1(cgroupMounts) {
		r.cgroupVersion = 1

		r.impl, err = newReaderV1(cgroupMounts, r.cgroupV1BaseController, r.readerFilter)
		if err != nil {
			return err
		}
	} else if isCgroup2(cgroupMounts) {
		r.cgroupVersion = 2

		r.impl, err = newReaderV2(cgroupMounts[cgroupV2Key], r.readerFilter)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unable to detect cgroup version from detected mount points: %v", cgroupMounts)
	}

	return nil
}

// ListCgroups returns list of known cgroups
func (r *Reader) ListCgroups() ([]Cgroup, error) {
	if err := r.refreshCache(); err != nil {
		return nil, err
	}

	cgroups := make([]Cgroup, 0, len(r.cgroups))
	for _, cg := range r.cgroups {
		cgroups = append(cgroups, cg)
	}

	return cgroups, nil
}

// GetCgroup returns cgroup for a given id, or nil if not found.
func (r *Reader) GetCgroup(id string) (Cgroup, error) {
	if err := r.refreshCache(); err != nil {
		return nil, err
	}

	return r.cgroups[id], nil
}

// RefreshCgroups forces the refresh of cgroup paths, regardless of cache settings.
func (r *Reader) RefreshCgroups() error {
	newCgroups, err := r.impl.parseCgroups()
	if err != nil {
		return err
	}

	r.scrapeTimestmap = time.Now()
	r.cgroups = newCgroups
	return nil
}

func (r *Reader) refreshCache() error {
	if r.cacheValidFor == 0 || time.Now().After(r.scrapeTimestmap.Add(r.cacheValidFor)) {
		return r.RefreshCgroups()
	}

	return nil
}
