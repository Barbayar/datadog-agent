// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package sampler

import (
	"math"
	"strconv"
	"sync"
	"time"
)

// DynamicConfig contains configuration items which may change
// dynamically over time.
type DynamicConfig struct {
	// RateByService contains the rate for each service/env tuple,
	// used in priority sampling by client libs.
	RateByService RateByService
}

// NewDynamicConfig creates a new dynamic config object which maps service signatures
// to their corresponding sampling rates. Each service will have a default assigned
// matching the service rate of the specified env.
func NewDynamicConfig(env string) *DynamicConfig {
	return &DynamicConfig{RateByService: RateByService{defaultEnv: env}}
}

// State TODO
type State struct {
	Rates      map[string]float64
	Mechanisms map[string]uint32
	Version    string
}

// RateByService stores the sampling rate per service. It is thread-safe, so
// one can read/write on it concurrently, using getters and setters.
type RateByService struct {
	defaultEnv string // env. to use for service defaults

	mu            sync.RWMutex // guards rates
	rates         *map[string]rm
	incomingRates *map[string]rm
	version       string
	rateHash      float64
}

// SetAll the sampling rate for all services. If a service/env is not
// in the map, then the entry is removed.
func (rbs *RateByService) SetAll(rates map[ServiceSignature]rm) {
	rbs.mu.Lock()
	defer rbs.mu.Unlock()

	changed := false
	if rbs.incomingRates == nil {
		m := make(map[string]rm, len(rates))
		rbs.incomingRates = &m
	}
	for k := range *rbs.incomingRates {
		delete(*rbs.incomingRates, k)
	}
	if rbs.rates == nil {
		changed = true
	}
	for k, v := range rates {
		ks := k.String()
		if !changed {
			r, ok := (*rbs.rates)[ks]
			if !ok || r != v {
				changed = true
			}
		}
		v.r = math.Min(math.Max(v.r, 0), 1)
		(*rbs.incomingRates)[ks] = v
		if k.Env == rbs.defaultEnv {
			// if this is the default env, then this is also the
			// service's default rate unbound to any env.
			(*rbs.incomingRates)[ServiceSignature{Name: k.Name}.String()] = v
		}
	}
	if len(*rbs.rates) != len(*rbs.incomingRates) {
		changed = true
	}
	if changed {
		tmp := rbs.rates
		rbs.rates = rbs.incomingRates
		rbs.incomingRates = tmp // we will reuse this map
		rbs.version = strconv.FormatInt(time.Now().Unix(), 16)
	}
}

// GetNewState returns the current state if the given version is different from the local version.
func (rbs *RateByService) GetNewState(version string) State {
	rbs.mu.RLock()
	defer rbs.mu.RUnlock()

	if rbs.version == version {
		return State{
			Version: version,
		}
	}
	ret := State{
		Rates:      make(map[string]float64, len(*rbs.rates)),
		Mechanisms: make(map[string]uint32, len(*rbs.rates)),
		Version:    rbs.version,
	}
	for k, v := range *rbs.rates {
		ret.Rates[k] = v.r
		if v.m != 0 {
			ret.Mechanisms[k] = v.m
		}
	}

	return ret
}
