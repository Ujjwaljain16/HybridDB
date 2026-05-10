package page_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ujjwaljain16/hybriddb/internal/storage/page"
)

func TestPageSnapshots(t *testing.T) {
	tests := []struct {
		name string
		setup func(*page.SlottedPage)
	}{
		{
			name: "empty_page",
			setup: func(sp *page.SlottedPage) {
				sp.Init(1)
			},
		},
		{
			name: "single_tuple",
			setup: func(sp *page.SlottedPage) {
				sp.Init(1)
				sp.InsertTuple([]byte("hello world"))
			},
		},
		{
			name: "fragmented_page",
			setup: func(sp *page.SlottedPage) {
				sp.Init(1)
				sp.InsertTuple([]byte("first"))
				sp.InsertTuple([]byte("second"))
				sp.DeleteTuple(0)
				sp.InsertTuple([]byte("third"))
			},
		},
		{
			name: "compacted_page",
			setup: func(sp *page.SlottedPage) {
				sp.Init(1)
				sp.InsertTuple([]byte("first"))
				sp.InsertTuple([]byte("second"))
				sp.DeleteTuple(0)
				sp.Compact()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw page.RawPage
			sp := page.NewSlottedPage(&raw)
			tc.setup(sp)

			snapshotPath := filepath.Join("testdata", "pages", tc.name + ".hex")
			
			// Get actual hex dump
			actualHex := hex.Dump(raw[:])

			// If update flag or file missing, create it
			if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
				err := os.WriteFile(snapshotPath, []byte(actualHex), 0644)
				if err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				t.Logf("Created golden file: %s", snapshotPath)
				return
			}

			// Read expected hex
			expectedHexBytes, err := os.ReadFile(snapshotPath)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			expectedHex := string(expectedHexBytes)

			if actualHex != expectedHex {
				t.Errorf("Snapshot mismatch for %s!\nIf this change was intentional, delete the .hex file and rerun tests to regenerate.\n", tc.name)
				// Log first difference for convenience
				if len(actualHex) > 512 {
					t.Logf("Actual (truncated):\n%s", actualHex[:512])
				}
			}
		})
	}
}
