package pager

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Ujjwaljain16/hybriddb/internal/common"
	"github.com/Ujjwaljain16/hybriddb/internal/config"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/page"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

type Pager interface {
	ReadPage(pageID uint64) (*page.RawPage, error)
	WritePage(pageID uint64, data *page.RawPage) error
	AllocatePage() (uint64, error)
	Sync() error
	Close() error
}

type DiskPager struct {
	file     *os.File
	numPages uint64
}

func NewDiskPager(filename string) (*DiskPager, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	size := info.Size()
	if size%config.PageSize != 0 {
		return nil, fmt.Errorf("file size is not a multiple of page size")
	}

	return &DiskPager{
		file:     file,
		numPages: uint64(size / config.PageSize),
	}, nil
}

func (p *DiskPager) ReadPage(pageID uint64) (*page.RawPage, error) {
	if pageID >= p.numPages {
		return nil, common.ErrPageNotFound
	}

	offset := int64(pageID) * config.PageSize
	var raw page.RawPage

	_, err := p.file.ReadAt(raw[:], offset)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, common.ErrPageNotFound
		}
		return nil, err
	}

	// Verify Header Integrity
	if string(raw[0:4]) != config.MagicBytes {
		return nil, fmt.Errorf("%w: invalid magic bytes", common.ErrPageCorrupted)
	}

	version := serialization.ReadUint16(raw[4:6])
	if version != config.FormatVersion {
		return nil, fmt.Errorf("%w: unsupported format version %d", common.ErrPageCorrupted, version)
	}

	// Verify Checksum
	storedChecksum := serialization.ReadUint32(raw[32:36])
	
	// Zero out checksum field for computation
	serialization.WriteUint32(raw[32:36], 0)
	
	// Checksum over 0-31 and 36-4095 is equivalent to checksum over the whole page 
	// with bytes 32-35 zeroed.
	computed := serialization.ComputeChecksum(raw[:])
	
	// Restore checksum field
	serialization.WriteUint32(raw[32:36], storedChecksum)

	if storedChecksum != computed {
		return nil, fmt.Errorf("%w: checksum mismatch", common.ErrPageCorrupted)
	}

	return &raw, nil
}

func (p *DiskPager) WritePage(pageID uint64, data *page.RawPage) error {
	// Recompute checksum before writing
	serialization.WriteUint32(data[32:36], 0)
	computed := serialization.ComputeChecksum(data[:])
	serialization.WriteUint32(data[32:36], computed)

	offset := int64(pageID) * config.PageSize
	_, err := p.file.WriteAt(data[:], offset)
	if err != nil {
		return err
	}
	
	if pageID >= p.numPages {
		p.numPages = pageID + 1
	}

	return nil
}

func (p *DiskPager) AllocatePage() (uint64, error) {
	pageID := p.numPages
	p.numPages++
	
	// Return a zero-initialized page, the caller is responsible for formatting it
	// and calling WritePage. However, to ensure the file grows, we write an empty 
	// block or we just let it grow on the next WritePage. It's safer to grow it now.
	
	var empty page.RawPage
	offset := int64(pageID) * config.PageSize
	_, err := p.file.WriteAt(empty[:], offset)
	if err != nil {
		p.numPages--
		return 0, err
	}
	
	return pageID, nil
}

func (p *DiskPager) Sync() error {
	return p.file.Sync()
}

func (p *DiskPager) Close() error {
	return p.file.Close()
}
