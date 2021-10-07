// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package pb

// ITrace TODO
type ITrace interface {
	GetSpans() []*Span
	GetBaggage() map[string]string
	Msgsize() int
}

// ITraces TODO
type ITraces interface {
	Len() int
	Get(i int) ITrace
}

//go:generate go run github.com/tinylib/msgp -file=span.pb.go -o span_gen.go -io=false
//go:generate go run github.com/tinylib/msgp -io=false

// Trace is a collection of spans with the same trace ID
type Trace []*Span

// GetSpans TODO
func (t Trace) GetSpans() []*Span {
	return t
}

// GetBaggage TODO
func (t Trace) GetBaggage() map[string]string {
	return nil
}

// Traces is a list of traces. This model matters as this is what we unpack from msgp.
type Traces []Trace

func (t Traces) Len() int {
	return len(t)
}

func (t Traces) Get(i int) ITrace {
	return t[i]
}

// TraceV6 TODO
type TraceV6 struct {
	Spans   []*Span
	Baggage map[string]string
}

// GetSpans TODO
func (t TraceV6) GetSpans() []*Span {
	return t.Spans
}

// GetBaggage TODO
func (t TraceV6) GetBaggage() map[string]string {
	return t.Baggage
}

type TracesV6 []*TraceV6

func (t TracesV6) Len() int {
	return len(t)
}

func (t TracesV6) Get(i int) ITrace {
	return t[i]
}

// TracesPayload TODO
type TracesPayload struct {
	TracerTags map[string]string
	Traces     TracesV6
}
