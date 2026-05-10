package pager

import "github.com/Ujjwaljain16/hybriddb/internal/common"

type Pager interface {
	ReadPage(pageID uint32) ([]byte, error)
	WritePage(pageID uint32, data []byte) error
	AllocatePage() (uint32, error)
	Sync() error
	Close() error
}

type DiskPager struct{}

func NewDiskPager(filename string) (*DiskPager, error) {
	panic("not implemented")
}

func (p *DiskPager) ReadPage(pageID uint32) ([]byte, error) {
	return nil, common.ErrNotImplemented
}

func (p *DiskPager) WritePage(pageID uint32, data []byte) error {
	return common.ErrNotImplemented
}

func (p *DiskPager) AllocatePage() (uint32, error) {
	return 0, common.ErrNotImplemented
}

func (p *DiskPager) Sync() error {
	return common.ErrNotImplemented
}

func (p *DiskPager) Close() error {
	return common.ErrNotImplemented
}
