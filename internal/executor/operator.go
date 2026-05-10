package executor

import (
	"github.com/Ujjwaljain16/hybriddb/internal/bufferpool"
	"github.com/Ujjwaljain16/hybriddb/internal/metrics"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/tuple"
	"github.com/Ujjwaljain16/hybriddb/internal/trace"
	"github.com/Ujjwaljain16/hybriddb/internal/wal"
)

type ExecContext struct {
	TxnID   uint32
	BufPool bufferpool.BufferPool
	WAL     wal.WALManager
	Metrics *metrics.MetricsCollector
	Tracer  *trace.ExecutionTracer
	QueryID string
}

type Operator interface {
	Open(ctx *ExecContext) error
	Next() (*tuple.Tuple, error)
	Close() error
}
