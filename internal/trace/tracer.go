package trace

import "time"

type EventType string

const (
	EventPageRead    EventType = "PageRead"
	EventPageWrite   EventType = "PageWrite"
	EventWALAppend   EventType = "WALAppend"
	EventNodeSplit   EventType = "NodeSplit"
	EventQueryStart  EventType = "QueryStart"
	EventQueryFinish EventType = "QueryFinish"
)

type TraceEvent struct {
	Timestamp    time.Time
	QueryID      string
	OperatorName string
	EventType    EventType
	Metadata     map[string]any
}

type ExecutionTracer struct{}

func NewTracer() *ExecutionTracer {
	return &ExecutionTracer{}
}

func (t *ExecutionTracer) RecordEvent(event TraceEvent) {
	panic("not implemented")
}
