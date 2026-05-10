package bufferpool

import "github.com/Ujjwaljain16/hybriddb/internal/common"

type BufferMetrics struct {
	HitRate     float64
	Evictions   uint64
	DirtyCount  int
	PinnedCount int
}

type BufferPool interface {
	GetPage(pageID uint32) ([]byte, error)
	UnpinPage(pageID uint32, isDirty bool) error
	FlushPage(pageID uint32) error
	FlushAll() error
	GetMetrics() BufferMetrics
}

type LRUBufferPool struct{}

func NewLRUBufferPool(numFrames int) *LRUBufferPool {
	panic("not implemented")
}

func (p *LRUBufferPool) GetPage(pageID uint32) ([]byte, error) {
	return nil, common.ErrNotImplemented
}

func (p *LRUBufferPool) UnpinPage(pageID uint32, isDirty bool) error {
	return common.ErrNotImplemented
}

func (p *LRUBufferPool) FlushPage(pageID uint32) error {
	return common.ErrNotImplemented
}

func (p *LRUBufferPool) FlushAll() error {
	return common.ErrNotImplemented
}

func (p *LRUBufferPool) GetMetrics() BufferMetrics {
	panic("not implemented")
}
