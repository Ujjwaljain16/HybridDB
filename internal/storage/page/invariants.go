package page

import (
	"fmt"

	"github.com/Ujjwaljain16/hybriddb/internal/config"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

// CheckInvariants performs strict validation of a slotted page's internal consistency.
func (p *SlottedPage) CheckInvariants() error {
	// 1. Header Validation
	if string(p.raw[0:4]) != config.MagicBytes {
		return fmt.Errorf("invalid magic bytes")
	}

	if serialization.ReadUint16(p.raw[6:8]) != uint16(PageTypeSlotted) {
		return fmt.Errorf("not a slotted page")
	}

	lower := p.getLowerBound()
	upper := p.getUpperBound()
	slotCount := p.GetSlotCount()

	// 2. Bound Validation
	expectedLower := uint16(config.PageHeaderSize) + uint16(slotCount*8)
	if lower != expectedLower {
		return fmt.Errorf("lower bound mismatch: %d != %d", lower, expectedLower)
	}

	if lower > upper {
		return fmt.Errorf("free space overlapped: lower %d > upper %d", lower, upper)
	}

	if upper > config.PageSize {
		return fmt.Errorf("upper bound out of bounds: %d", upper)
	}

	// 3. Slot Validation (Overlap detection)
	// To detect overlap, we track used regions. Since tuples could be out of order,
	// we just sort or check against each other.
	// For O(N^2) it's fine since N is small, or we can use a boolean array.
	used := make([]bool, config.PageSize)

	for i := uint32(0); i < slotCount; i++ {
		slotOffset := config.PageHeaderSize + uint16(i*8)
		flags := serialization.ReadUint16(p.raw[slotOffset+4 : slotOffset+6])
		
		if flags&SlotFlagDeleted == 0 {
			tupleOff := serialization.ReadUint16(p.raw[slotOffset : slotOffset+2])
			tupleLen := serialization.ReadUint16(p.raw[slotOffset+2 : slotOffset+4])

			if tupleOff < upper || tupleOff+tupleLen > config.PageSize {
				return fmt.Errorf("slot %d points outside tuple region [%d, %d]", i, tupleOff, tupleOff+tupleLen)
			}

			// Check overlap
			for j := tupleOff; j < tupleOff+tupleLen; j++ {
				if used[j] {
					return fmt.Errorf("tuple overlap detected at byte %d", j)
				}
				used[j] = true
			}
		}
	}

	return nil
}
