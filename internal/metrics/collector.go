package metrics

type QueryMetrics struct {
	TotalTimeMs int64
}

type OperatorStats struct {
	OperatorName string
	RowsIn       uint64
	RowsOut      uint64
	PagesRead    uint64
	TimeMs       int64
}

type MetricsCollector struct{}

func NewCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (m *MetricsCollector) RecordQuery(q *QueryMetrics) {
	panic("not implemented")
}

func (m *MetricsCollector) RecordOperator(s *OperatorStats) {
	panic("not implemented")
}
