// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/api/apiutil"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	receiverErrorKey = "datadog.trace_agent.receiver.error"
)

// We encaspulate the answers in a container, this is to ease-up transition,
// should we add another fied.
type traceResponse struct {
	// All the sampling rates recommended, by service
	Rates map[string]float64 `json:"rate_by_service"`
}

// httpFormatError is used for payload format errors
func httpFormatError(w http.ResponseWriter, v Version, err error) {
	log.Errorf("Rejecting client request: %v", err)
	tags := []string{"error:format-error", "version:" + string(v)}
	metrics.Count(receiverErrorKey, 1, tags, 1)
	http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
}

// httpDecodingError is used for errors happening in decoding
func httpDecodingError(err error, tags []string, w http.ResponseWriter) {
	status := http.StatusBadRequest
	errtag := "decoding-error"
	msg := err.Error()

	switch err {
	case apiutil.ErrLimitedReaderLimitReached:
		status = http.StatusRequestEntityTooLarge
		errtag = "payload-too-large"
		msg = errtag
	case io.EOF, io.ErrUnexpectedEOF:
		errtag = "unexpected-eof"
		msg = errtag
	}
	if err, ok := err.(net.Error); ok && err.Timeout() {
		status = http.StatusRequestTimeout
		errtag = "timeout"
		msg = errtag
	}

	tags = append(tags, fmt.Sprintf("error:%s", errtag))
	metrics.Count(receiverErrorKey, 1, tags, 1)
	http.Error(w, msg, status)
}

// httpOK is a dumb response for when things are a OK. It returns the number
// of bytes written along with a boolean specifying if the response was successful.
func httpOK(w http.ResponseWriter) (n uint64, ok bool) {
	nn, err := io.WriteString(w, "OK\n")
	return uint64(nn), err == nil
}

type writeCounter struct {
	w io.Writer
	n uint64
}

func newWriteCounter(w io.Writer) *writeCounter {
	return &writeCounter{w: w}
}

func (wc *writeCounter) Write(p []byte) (n int, err error) {
	atomic.AddUint64(&wc.n, uint64(len(p)))
	return wc.w.Write(p)
}

func (wc *writeCounter) N() uint64 { return atomic.LoadUint64(&wc.n) }

// httpRateByService outputs, as a JSON, the recommended sampling rates for all services.
// It returns the number of bytes written and a boolean specifying whether the write was
// successful.
func httpRateByService(w http.ResponseWriter, dynConf *sampler.DynamicConfig) (n uint64, ok bool) {
	w.Header().Set("Content-Type", "application/json")
	response := traceResponse{
		Rates: dynConf.RateByService.GetAll(), // this is thread-safe
	}
	wc := newWriteCounter(w)
	ok = true
	encoder := json.NewEncoder(wc)
	if err := encoder.Encode(response); err != nil {
		tags := []string{"error:response-error"}
		metrics.Count(receiverErrorKey, 1, tags, 1)
		ok = false
	}
	return wc.N(), ok
}
