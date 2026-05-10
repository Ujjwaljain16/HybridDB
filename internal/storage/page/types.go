package page

import "github.com/Ujjwaljain16/hybriddb/internal/config"

type PageType uint16

const (
	PageTypeFree          PageType = 0x0000
	PageTypeSlotted       PageType = 0x0001
	PageTypeBTreeInternal PageType = 0x0002
	PageTypeBTreeLeaf     PageType = 0x0003
)

// RawPage is the raw byte array interpretation of a page on disk.
type RawPage [config.PageSize]byte

// RID (Record Identifier) uniquely identifies a tuple.
type RID struct {
	PageID    uint64
	SlotIndex uint32
}
