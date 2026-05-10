package serialization

import (
	"encoding/binary"
	"hash/crc32"
	"math"
)

var (
	// Endian defines the strict byte order used throughout HybridDB.
	Endian = binary.LittleEndian
)

// ChecksumTable uses the Castagnoli polynomial (CRC32C)
var ChecksumTable = crc32.MakeTable(crc32.Castagnoli)

// ComputeChecksum calculates the CRC32C over the provided data.
func ComputeChecksum(data []byte) uint32 {
	return crc32.Checksum(data, ChecksumTable)
}

func WriteUint16(b []byte, v uint16) {
	Endian.PutUint16(b, v)
}

func ReadUint16(b []byte) uint16 {
	return Endian.Uint16(b)
}

func WriteUint32(b []byte, v uint32) {
	Endian.PutUint32(b, v)
}

func ReadUint32(b []byte) uint32 {
	return Endian.Uint32(b)
}

func WriteUint64(b []byte, v uint64) {
	Endian.PutUint64(b, v)
}

func ReadUint64(b []byte) uint64 {
	return Endian.Uint64(b)
}

func WriteInt32(b []byte, v int32) {
	Endian.PutUint32(b, uint32(v))
}

func ReadInt32(b []byte) int32 {
	return int32(Endian.Uint32(b))
}

func WriteInt64(b []byte, v int64) {
	Endian.PutUint64(b, uint64(v))
}

func ReadInt64(b []byte) int64 {
	return int64(Endian.Uint64(b))
}

func WriteFloat32(b []byte, v float32) {
	Endian.PutUint32(b, math.Float32bits(v))
}

func ReadFloat32(b []byte) float32 {
	return math.Float32frombits(Endian.Uint32(b))
}

func WriteVector(b []byte, vec []float32) {
	offset := 0
	for _, v := range vec {
		WriteFloat32(b[offset:offset+4], v)
		offset += 4
	}
}

func ReadVector(b []byte, dims uint32) []float32 {
	vec := make([]float32, dims)
	offset := 0
	for i := uint32(0); i < dims; i++ {
		vec[i] = ReadFloat32(b[offset : offset+4])
		offset += 4
	}
	return vec
}
