package debug

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Ujjwaljain16/hybriddb/internal/config"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/page"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

// HexDump returns an xxd-style string dump of the data.
func HexDump(data []byte) string {
	return hex.Dump(data)
}

// ExplainPage interprets a raw 4KB page and returns a human-readable summary of its internal state.
func ExplainPage(data []byte) string {
	if len(data) != config.PageSize {
		return fmt.Sprintf("Invalid page size: expected %d, got %d", config.PageSize, len(data))
	}

	var sb strings.Builder

	magic := string(data[0:4])
	version := serialization.ReadUint16(data[4:6])
	pageType := serialization.ReadUint16(data[6:8])
	pageID := serialization.ReadUint64(data[8:16])
	lsn := serialization.ReadUint64(data[16:24])
	lowerBound := serialization.ReadUint16(data[24:26])
	upperBound := serialization.ReadUint16(data[26:28])
	slotCount := serialization.ReadUint32(data[28:32])
	checksum := serialization.ReadUint32(data[32:36])

	sb.WriteString(fmt.Sprintf("Page ID:       %d\n", pageID))
	sb.WriteString(fmt.Sprintf("Magic:         %s\n", magic))
	sb.WriteString(fmt.Sprintf("Version:       %d\n", version))
	
	typeStr := "Unknown"
	switch page.PageType(pageType) {
	case page.PageTypeFree:
		typeStr = "Free"
	case page.PageTypeSlotted:
		typeStr = "Slotted"
	case page.PageTypeBTreeInternal:
		typeStr = "BTreeInternal"
	case page.PageTypeBTreeLeaf:
		typeStr = "BTreeLeaf"
	}
	
	sb.WriteString(fmt.Sprintf("Type:          %s (0x%04x)\n", typeStr, pageType))
	sb.WriteString(fmt.Sprintf("LSN:           %d\n", lsn))
	sb.WriteString(fmt.Sprintf("Lower Bound:   %d (Slot End)\n", lowerBound))
	sb.WriteString(fmt.Sprintf("Upper Bound:   %d (Tuple Start)\n", upperBound))
	if upperBound >= lowerBound {
		sb.WriteString(fmt.Sprintf("Free Space:    %d bytes\n", upperBound-lowerBound))
	} else {
		sb.WriteString(fmt.Sprintf("Free Space:    OVERLAPPED! (%d)\n", int(upperBound)-int(lowerBound)))
	}
	sb.WriteString(fmt.Sprintf("Slot Count:    %d\n", slotCount))
	sb.WriteString(fmt.Sprintf("Checksum:      0x%08x\n", checksum))
	sb.WriteString("----------------------------------------\n")
	
	if page.PageType(pageType) == page.PageTypeSlotted {
		sb.WriteString("Slots:\n")
		for i := uint32(0); i < slotCount; i++ {
			slotOffset := uint16(config.PageHeaderSize) + uint16(i*8)
			tupleOff := serialization.ReadUint16(data[slotOffset : slotOffset+2])
			tupleLen := serialization.ReadUint16(data[slotOffset+2 : slotOffset+4])
			flags := serialization.ReadUint16(data[slotOffset+4 : slotOffset+6])
			
			status := "active"
			if flags&page.SlotFlagDeleted != 0 {
				status = "deleted"
			} else if flags&page.SlotFlagRedirected != 0 {
				status = "redirected"
			}
			
			sb.WriteString(fmt.Sprintf("  [%d] offset=%d len=%d status=%s\n", i, tupleOff, tupleLen, status))
		}
	}

	return sb.String()
}
