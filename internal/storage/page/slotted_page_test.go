package page_test

import (
	"bytes"
	"testing"

	"github.com/Ujjwaljain16/hybriddb/internal/storage/page"
	"pgregory.net/rapid"
)

func TestSlottedPageProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var raw page.RawPage
		sp := page.NewSlottedPage(&raw)
		sp.Init(42)

		type tupleRecord struct {
			idx  uint32
			data []byte
		}

		active := make(map[uint32][]byte)
		var records []tupleRecord

		// Number of operations
		ops := rapid.IntRange(1, 100).Draw(t, "ops")

		for i := 0; i < ops; i++ {
			// Choose operation: Insert(0), Delete(1), Compact(2)
			op := rapid.IntRange(0, 2).Draw(t, "op")

			switch op {
			case 0: // Insert
				// Data size between 1 and 100 bytes
				dataSize := rapid.IntRange(1, 100).Draw(t, "dataSize")
				data := rapid.SliceOfN(rapid.Byte(), dataSize, dataSize).Draw(t, "data")
				
				idx, err := sp.InsertTuple(data)
				if err == nil {
					active[idx] = data
					records = append(records, tupleRecord{idx, data})
				}
			case 1: // Delete
				if len(records) > 0 {
					recordIdx := rapid.IntRange(0, len(records)-1).Draw(t, "recordIdx")
					tr := records[recordIdx]
					sp.DeleteTuple(tr.idx)
					delete(active, tr.idx)
					// Remove from records slice
					records[recordIdx] = records[len(records)-1]
					records = records[:len(records)-1]
				}
			case 2: // Compact
				sp.Compact()
			}

			// Validate Invariants after every operation
			if err := sp.CheckInvariants(); err != nil {
				t.Fatalf("invariants failed: %v", err)
			}

			// Validate all active tuples
			for idx, expectedData := range active {
				actualData, err := sp.GetTuple(idx)
				if err != nil {
					t.Fatalf("failed to get active tuple %d: %v", idx, err)
				}
				if !bytes.Equal(actualData, expectedData) {
					t.Fatalf("tuple %d data mismatch", idx)
				}
			}
		}
	})
}
