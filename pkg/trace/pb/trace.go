// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package pb

//go:generate go run github.com/tinylib/msgp -file=span.pb.go -o span_gen.go -io=false
//go:generate go run github.com/tinylib/msgp -io=false

// Trace is a collection of spans with the same trace ID
type Trace []*Span

// Traces is a list of traces. This model matters as this is what we unpack from msgp.
type Traces []Trace

// TraceChunk is a collection of spans with the same trace ID and their baggage
type TraceChunk struct {
	Priority int32
	Spans    []*Span
}

// TracerPayload is a collection of trace chunks and tracer tags
type TracerPayload struct {
	Tags   map[string]string
	Chunks []*TraceChunk
}
