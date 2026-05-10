package wal

import "github.com/Ujjwaljain16/hybriddb/internal/common"

type LSN uint64

type RecordType uint8

const (
	RecordTypeInsert RecordType = iota
	RecordTypeUpdate
	RecordTypeDelete
	RecordTypeCheckpoint
)

type Record struct {
	LSN    LSN
	Type   RecordType
	TxnID  uint32
	PageID uint32
	Data   []byte
}

type WALManager interface {
	AppendRecord(rec *Record) (LSN, error)
	EnsureFlushed(lsn LSN) error
	Checkpoint() error
	Recover() error
}

type DiskWALManager struct{}

func NewDiskWALManager(filename string) (*DiskWALManager, error) {
	panic("not implemented")
}

func (w *DiskWALManager) AppendRecord(rec *Record) (LSN, error) {
	return 0, common.ErrNotImplemented
}

func (w *DiskWALManager) EnsureFlushed(lsn LSN) error {
	return common.ErrNotImplemented
}

func (w *DiskWALManager) Checkpoint() error {
	return common.ErrNotImplemented
}

func (w *DiskWALManager) Recover() error {
	return common.ErrNotImplemented
}
