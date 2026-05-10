package page

import (
	"fmt"

	"github.com/Ujjwaljain16/hybriddb/internal/common"
	"github.com/Ujjwaljain16/hybriddb/internal/config"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

// SlottedPage is a logical wrapper around a RawPage.
// It manages the slot array and tuple data regions.
type SlottedPage struct {
	raw *RawPage
}

func NewSlottedPage(raw *RawPage) *SlottedPage {
	return &SlottedPage{raw: raw}
}

// Initizes the page as a clean SlottedPage.
func (p *SlottedPage) Init(pageID uint64) {
	// 1. Zero all bytes
	for i := range p.raw {
		p.raw[i] = 0
	}

	// 2. Initialize header fields
	copy(p.raw[0:4], config.MagicBytes)
	serialization.WriteUint16(p.raw[4:6], config.FormatVersion)
	serialization.WriteUint16(p.raw[6:8], uint16(PageTypeSlotted))
	serialization.WriteUint64(p.raw[8:16], pageID)

	// 3. Initialize boundaries
	serialization.WriteUint16(p.raw[24:26], config.PageHeaderSize) // Lower Free Space Bound (Slot Array End)
	serialization.WriteUint16(p.raw[26:28], config.PageSize)       // Upper Free Space Bound (Tuple Data Start)
	serialization.WriteUint32(p.raw[28:32], 0)                     // Slot Count
}

func (p *SlottedPage) GetSlotCount() uint32 {
	return serialization.ReadUint32(p.raw[28:32])
}

func (p *SlottedPage) setSlotCount(count uint32) {
	serialization.WriteUint32(p.raw[28:32], count)
}

func (p *SlottedPage) getLowerBound() uint16 {
	return serialization.ReadUint16(p.raw[24:26])
}

func (p *SlottedPage) setLowerBound(bound uint16) {
	serialization.WriteUint16(p.raw[24:26], bound)
}

func (p *SlottedPage) getUpperBound() uint16 {
	return serialization.ReadUint16(p.raw[26:28])
}

func (p *SlottedPage) setUpperBound(bound uint16) {
	serialization.WriteUint16(p.raw[26:28], bound)
}

func (p *SlottedPage) GetFreeSpace() uint16 {
	return p.getUpperBound() - p.getLowerBound()
}

// Slot Flags
const (
	SlotFlagActive     uint16 = 0x0000
	SlotFlagDeleted    uint16 = 0x0001
	SlotFlagRedirected uint16 = 0x0002
)

// InsertTuple adds a tuple to the page and returns its SlotIndex.
func (p *SlottedPage) InsertTuple(data []byte) (uint32, error) {
	dataLen := uint16(len(data))
	if dataLen > config.PageSize-config.PageHeaderSize {
		return 0, fmt.Errorf("tuple too large")
	}

	// Calculate space required: 8 bytes for new slot + data length
	// unless we can reuse a deleted slot, which saves the 8 bytes.
	freeSpace := p.GetFreeSpace()
	
	// 1. Try to find a deleted slot
	var slotIdx uint32 = mathMaxUint32
	slotCount := p.GetSlotCount()
	for i := uint32(0); i < slotCount; i++ {
		slotOffset := config.PageHeaderSize + (i * 8)
		flags := serialization.ReadUint16(p.raw[slotOffset+4 : slotOffset+6])
		if flags&SlotFlagDeleted != 0 {
			slotIdx = i
			break
		}
	}

	spaceNeeded := dataLen
	if slotIdx == mathMaxUint32 {
		spaceNeeded += 8 // new slot entry
	}

	if freeSpace < spaceNeeded {
		return 0, common.ErrBufferPoolFull // or a new ErrPageFull
	}

	// 2. Allocate data at the upper bound
	newUpperBound := p.getUpperBound() - dataLen
	copy(p.raw[newUpperBound:p.getUpperBound()], data)
	p.setUpperBound(newUpperBound)

	// 3. Write slot entry
	if slotIdx == mathMaxUint32 {
		// New slot
		slotIdx = slotCount
		p.setSlotCount(slotCount + 1)
		slotOffset := p.getLowerBound()
		p.setLowerBound(slotOffset + 8)
		
		serialization.WriteUint16(p.raw[slotOffset:slotOffset+2], newUpperBound)
		serialization.WriteUint16(p.raw[slotOffset+2:slotOffset+4], dataLen)
		serialization.WriteUint16(p.raw[slotOffset+4:slotOffset+6], SlotFlagActive)
		serialization.WriteUint16(p.raw[slotOffset+6:slotOffset+8], 0) // Reserved
	} else {
		// Reuse slot
		slotOffset := config.PageHeaderSize + uint16(slotIdx*8)
		serialization.WriteUint16(p.raw[slotOffset:slotOffset+2], newUpperBound)
		serialization.WriteUint16(p.raw[slotOffset+2:slotOffset+4], dataLen)
		serialization.WriteUint16(p.raw[slotOffset+4:slotOffset+6], SlotFlagActive)
	}

	return slotIdx, nil
}

func (p *SlottedPage) GetTuple(slotIdx uint32) ([]byte, error) {
	if slotIdx >= p.GetSlotCount() {
		return nil, fmt.Errorf("invalid slot index")
	}

	slotOffset := config.PageHeaderSize + uint16(slotIdx*8)
	flags := serialization.ReadUint16(p.raw[slotOffset+4 : slotOffset+6])
	
	if flags&SlotFlagDeleted != 0 {
		return nil, fmt.Errorf("tuple deleted")
	}

	tupleOff := serialization.ReadUint16(p.raw[slotOffset : slotOffset+2])
	tupleLen := serialization.ReadUint16(p.raw[slotOffset+2 : slotOffset+4])

	// Strict bounds checking
	if tupleOff < p.getLowerBound() || tupleOff+tupleLen > config.PageSize {
		return nil, fmt.Errorf("tuple bounds corrupted")
	}

	return p.raw[tupleOff : tupleOff+tupleLen], nil
}

func (p *SlottedPage) DeleteTuple(slotIdx uint32) error {
	if slotIdx >= p.GetSlotCount() {
		return fmt.Errorf("invalid slot index")
	}

	slotOffset := config.PageHeaderSize + uint16(slotIdx*8)
	flags := serialization.ReadUint16(p.raw[slotOffset+4 : slotOffset+6])
	
	if flags&SlotFlagDeleted != 0 {
		return nil // already deleted
	}

	// Mark as deleted. The actual bytes will be reclaimed on Compact()
	serialization.WriteUint16(p.raw[slotOffset+4:slotOffset+6], flags|SlotFlagDeleted)
	return nil
}

// Compact shifts all active tuple data to the upper end of the page to create a single 
// contiguous free space region. Crucially, Slot IDs remain completely stable.
func (p *SlottedPage) Compact() {
	slotCount := p.GetSlotCount()
	
	// Create a temporary buffer for the new tuple data area
	newData := make([]byte, config.PageSize)
	newUpperBound := uint16(config.PageSize)

	// We iterate through slots and copy active tuples to the top of newData
	for i := uint32(0); i < slotCount; i++ {
		slotOffset := config.PageHeaderSize + uint16(i*8)
		flags := serialization.ReadUint16(p.raw[slotOffset+4 : slotOffset+6])
		
		if flags&SlotFlagDeleted == 0 {
			tupleOff := serialization.ReadUint16(p.raw[slotOffset : slotOffset+2])
			tupleLen := serialization.ReadUint16(p.raw[slotOffset+2 : slotOffset+4])
			
			// Copy data to new location
			newUpperBound -= tupleLen
			copy(newData[newUpperBound:newUpperBound+tupleLen], p.raw[tupleOff:tupleOff+tupleLen])
			
			// Update slot offset
			serialization.WriteUint16(p.raw[slotOffset:slotOffset+2], newUpperBound)
		}
	}

	// Copy the compacted data region back to the raw page
	copy(p.raw[newUpperBound:config.PageSize], newData[newUpperBound:config.PageSize])
	
	// Zero out the newly freed space
	lowerBound := p.getLowerBound()
	for i := lowerBound; i < newUpperBound; i++ {
		p.raw[i] = 0
	}

	p.setUpperBound(newUpperBound)
}

const mathMaxUint32 = ^uint32(0)
