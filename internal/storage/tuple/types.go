package tuple

// TypeTag defines the serialized representation of tuple data types.
//
// WARNING:
// TypeTag numeric values are part of on-disk serialization.
// NEVER reorder or modify existing values.
type TypeTag uint8

const (
	TypeINT32   TypeTag = 0x01
	TypeINT64   TypeTag = 0x02
	TypeFLOAT32 TypeTag = 0x03
	TypeVARCHAR TypeTag = 0x04
	TypeVECTOR  TypeTag = 0x05
	TypeNULL    TypeTag = 0x06
	TypeBOOL    TypeTag = 0x07
)
