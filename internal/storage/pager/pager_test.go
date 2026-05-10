package pager_test

import (
	"errors"
	"os"
	"testing"

	"github.com/Ujjwaljain16/hybriddb/internal/common"
	"github.com/Ujjwaljain16/hybriddb/internal/config"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/page"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/pager"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

func createTestPager(t *testing.T) (pager.Pager, string) {
	f, err := os.CreateTemp("", "hybriddb-pager-test-*.hdb")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()
	
	p, err := pager.NewDiskPager(f.Name())
	if err != nil {
		t.Fatalf("failed to open pager: %v", err)
	}
	
	return p, f.Name()
}

func initRawPage(pageID uint64) *page.RawPage {
	var raw page.RawPage
	copy(raw[0:4], config.MagicBytes)
	serialization.WriteUint16(raw[4:6], config.FormatVersion)
	serialization.WriteUint16(raw[6:8], uint16(page.PageTypeSlotted))
	serialization.WriteUint64(raw[8:16], pageID)
	// free space boundaries
	serialization.WriteUint16(raw[24:26], config.PageHeaderSize)
	serialization.WriteUint16(raw[26:28], config.PageSize)
	return &raw
}

func TestPagerReadWrite(t *testing.T) {
	p, filename := createTestPager(t)
	defer os.Remove(filename)
	defer p.Close()

	pageID, err := p.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage failed: %v", err)
	}

	raw := initRawPage(pageID)
	// Add some dummy payload
	raw[100] = 42

	if err := p.WritePage(pageID, raw); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}

	readRaw, err := p.ReadPage(pageID)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	if readRaw[100] != 42 {
		t.Fatalf("expected 42 at offset 100, got %v", readRaw[100])
	}
}

func TestPagerCorruptionMatrix(t *testing.T) {
	p, filename := createTestPager(t)
	defer os.Remove(filename)

	pageID, _ := p.AllocatePage()
	raw := initRawPage(pageID)
	p.WritePage(pageID, raw)
	p.Close()

	tests := []struct {
		name     string
		corrupt  func(*os.File)
	}{
		{
			name: "Corrupt Magic Bytes",
			corrupt: func(f *os.File) {
				f.WriteAt([]byte("BAD!"), 0)
			},
		},
		{
			name: "Corrupt Version",
			corrupt: func(f *os.File) {
				f.WriteAt([]byte{0x99, 0x99}, 4)
			},
		},
		{
			name: "Corrupt Checksum",
			corrupt: func(f *os.File) {
				f.WriteAt([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 32)
			},
		},
		{
			name: "Corrupt Payload",
			corrupt: func(f *os.File) {
				f.WriteAt([]byte{0xFF}, 100) // Mutate payload after checksum was computed
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := os.OpenFile(filename, os.O_RDWR, 0666)
			tc.corrupt(f)
			f.Close()

			p2, _ := pager.NewDiskPager(filename)
			_, err := p2.ReadPage(pageID)
			p2.Close()

			if err == nil {
				t.Fatalf("expected error due to corruption, got nil")
			}
			if !errors.Is(err, common.ErrPageCorrupted) {
				t.Fatalf("expected ErrPageCorrupted, got %v", err)
			}
			
			// Repair for next test
			p3, _ := pager.NewDiskPager(filename)
			raw := initRawPage(pageID)
			p3.WritePage(pageID, raw)
			p3.Close()
		})
	}
}
