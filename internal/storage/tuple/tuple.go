package tuple

import (
	"fmt"
	"github.com/Ujjwaljain16/hybriddb/internal/common"
	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
)

type Column struct {
	Name string
	Type TypeTag
}

type Schema struct {
	Columns []Column
	Version uint16
}

type Value struct {
	Type   TypeTag
	IsNull bool
	
	ValInt32  int32
	ValInt64  int64
	ValFloat  float32
	ValStr    string
	ValVec    []float32
	ValBool   bool
}

type Tuple struct {
	Schema  *Schema
	Values  []Value
}

// Serialize encodes a tuple exactly according to docs/storage-format.md
func (t *Tuple) Serialize() ([]byte, error) {
	if len(t.Values) != len(t.Schema.Columns) {
		return nil, common.ErrInvalidTuple
	}

	colCount := len(t.Values)
	if colCount > 65535 {
		return nil, fmt.Errorf("too many columns")
	}

	// 1. Calculate Sizes
	nullBitmapSize := (colCount + 7) / 8
	fixedDataSize := 0
	varDataSize := 0

	for i, val := range t.Values {
		if val.Type != t.Schema.Columns[i].Type {
			return nil, common.ErrInvalidTuple
		}
		
		switch val.Type {
		case TypeINT32, TypeFLOAT32:
			fixedDataSize += 4
		case TypeINT64:
			fixedDataSize += 8
		case TypeBOOL:
			fixedDataSize += 1
		case TypeVARCHAR:
			fixedDataSize += 8 // 4 byte offset + 4 byte length
			if !val.IsNull {
				varDataSize += len(val.ValStr)
			}
		case TypeVECTOR:
			fixedDataSize += 8 // 4 byte offset + 4 byte length
			if !val.IsNull {
				varDataSize += 4 + len(val.ValVec)*4 // uint32 dims + float array
			}
		case TypeNULL:
			// no space needed, tracked in bitmap
		}
	}

	headerSize := 16
	totalLength := headerSize + fixedDataSize + nullBitmapSize + varDataSize
	
	buf := make([]byte, totalLength)
	
	// Write Header
	serialization.WriteUint32(buf[0:4], uint32(totalLength))
	serialization.WriteUint16(buf[4:6], uint16(colCount))
	serialization.WriteUint16(buf[6:8], uint16(headerSize+fixedDataSize)) // NullBitmapOffset
	serialization.WriteUint16(buf[8:10], uint16(headerSize+fixedDataSize+nullBitmapSize)) // VariableDataOffset
	serialization.WriteUint16(buf[10:12], t.Schema.Version)
	// buf[12:16] is reserved (zeroes)

	fixedOffset := headerSize
	varOffset := headerSize + fixedDataSize + nullBitmapSize
	nullOffset := headerSize + fixedDataSize

	for i, val := range t.Values {
		// Set Null Bitmap (LSB first)
		if val.IsNull {
			byteIdx := i / 8
			bitIdx := i % 8
			buf[nullOffset+byteIdx] |= (1 << bitIdx)
			// Skip data writing if null for fixed/var types, zeroes remain
		}

		// Write Data
		switch val.Type {
		case TypeINT32:
			if !val.IsNull {
				serialization.WriteInt32(buf[fixedOffset:fixedOffset+4], val.ValInt32)
			}
			fixedOffset += 4
		case TypeINT64:
			if !val.IsNull {
				serialization.WriteInt64(buf[fixedOffset:fixedOffset+8], val.ValInt64)
			}
			fixedOffset += 8
		case TypeFLOAT32:
			if !val.IsNull {
				serialization.WriteFloat32(buf[fixedOffset:fixedOffset+4], val.ValFloat)
			}
			fixedOffset += 4
		case TypeBOOL:
			if !val.IsNull && val.ValBool {
				buf[fixedOffset] = 1
			}
			fixedOffset += 1
		case TypeVARCHAR:
			if !val.IsNull {
				strLen := len(val.ValStr)
				serialization.WriteUint32(buf[fixedOffset:fixedOffset+4], uint32(varOffset))
				serialization.WriteUint32(buf[fixedOffset+4:fixedOffset+8], uint32(strLen))
				copy(buf[varOffset:varOffset+strLen], []byte(val.ValStr))
				varOffset += strLen
			}
			fixedOffset += 8
		case TypeVECTOR:
			if !val.IsNull {
				vecLen := len(val.ValVec)
				byteLen := 4 + vecLen*4
				serialization.WriteUint32(buf[fixedOffset:fixedOffset+4], uint32(varOffset))
				serialization.WriteUint32(buf[fixedOffset+4:fixedOffset+8], uint32(byteLen))
				
				serialization.WriteUint32(buf[varOffset:varOffset+4], uint32(vecLen))
				serialization.WriteVector(buf[varOffset+4:varOffset+byteLen], val.ValVec)
				varOffset += byteLen
			}
			fixedOffset += 8
		}
	}

	return buf, nil
}

// Deserialize parses a tuple from bytes according to the given schema
func Deserialize(data []byte, schema *Schema) (*Tuple, error) {
	if len(data) < 16 {
		return nil, common.ErrInvalidTuple
	}

	totalLength := serialization.ReadUint32(data[0:4])
	if totalLength != uint32(len(data)) {
		return nil, common.ErrInvalidTuple
	}

	colCount := serialization.ReadUint16(data[4:6])
	if colCount != uint16(len(schema.Columns)) {
		return nil, common.ErrInvalidTuple
	}

	nullBitmapOffset := serialization.ReadUint16(data[6:8])
	_ = serialization.ReadUint16(data[8:10]) // var data offset
	version := serialization.ReadUint16(data[10:12])

	if version != schema.Version {
		return nil, fmt.Errorf("schema version mismatch")
	}

	values := make([]Value, colCount)
	fixedOffset := 16 // right after header

	for i := 0; i < int(colCount); i++ {
		byteIdx := i / 8
		bitIdx := i % 8
		isNull := (data[nullBitmapOffset+uint16(byteIdx)] & (1 << bitIdx)) != 0
		colType := schema.Columns[i].Type
		
		val := Value{Type: colType, IsNull: isNull}

		switch colType {
		case TypeINT32:
			if !isNull {
				val.ValInt32 = serialization.ReadInt32(data[fixedOffset:fixedOffset+4])
			}
			fixedOffset += 4
		case TypeINT64:
			if !isNull {
				val.ValInt64 = serialization.ReadInt64(data[fixedOffset:fixedOffset+8])
			}
			fixedOffset += 8
		case TypeFLOAT32:
			if !isNull {
				val.ValFloat = serialization.ReadFloat32(data[fixedOffset:fixedOffset+4])
			}
			fixedOffset += 4
		case TypeBOOL:
			if !isNull {
				val.ValBool = data[fixedOffset] == 1
			}
			fixedOffset += 1
		case TypeVARCHAR:
			if !isNull {
				varOff := serialization.ReadUint32(data[fixedOffset:fixedOffset+4])
				varLen := serialization.ReadUint32(data[fixedOffset+4:fixedOffset+8])
				val.ValStr = string(data[varOff : varOff+varLen])
			}
			fixedOffset += 8
		case TypeVECTOR:
			if !isNull {
				varOff := serialization.ReadUint32(data[fixedOffset:fixedOffset+4])
				varLen := serialization.ReadUint32(data[fixedOffset+4:fixedOffset+8])
				
				dims := serialization.ReadUint32(data[varOff:varOff+4])
				if dims*4+4 != varLen {
					return nil, common.ErrInvalidTuple
				}
				val.ValVec = serialization.ReadVector(data[varOff+4:varOff+varLen], dims)
			}
			fixedOffset += 8
		}
		values[i] = val
	}

	return &Tuple{Schema: schema, Values: values}, nil
}
