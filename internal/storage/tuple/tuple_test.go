package tuple_test

import (
	"math"
	"testing"

	"github.com/Ujjwaljain16/hybriddb/internal/storage/tuple"
	"pgregory.net/rapid"
)

func genSchema(t *rapid.T) *tuple.Schema {
	numCols := rapid.IntRange(1, 20).Draw(t, "numCols")
	cols := make([]tuple.Column, numCols)
	for i := 0; i < numCols; i++ {
		types := []tuple.TypeTag{
			tuple.TypeINT32, tuple.TypeINT64, tuple.TypeFLOAT32,
			tuple.TypeVARCHAR, tuple.TypeVECTOR, tuple.TypeBOOL,
		}
		typ := rapid.SampledFrom(types).Draw(t, "type")
		cols[i] = tuple.Column{Name: "col", Type: typ}
	}
	return &tuple.Schema{Columns: cols, Version: 1}
}

func genTupleForSchema(t *rapid.T, schema *tuple.Schema) *tuple.Tuple {
	vals := make([]tuple.Value, len(schema.Columns))
	for i, col := range schema.Columns {
		isNull := rapid.Bool().Draw(t, "isNull")
		val := tuple.Value{Type: col.Type, IsNull: isNull}
		if !isNull {
			switch col.Type {
			case tuple.TypeINT32:
				val.ValInt32 = rapid.Int32().Draw(t, "int32")
			case tuple.TypeINT64:
				val.ValInt64 = rapid.Int64().Draw(t, "int64")
			case tuple.TypeFLOAT32:
				val.ValFloat = rapid.Float32().Draw(t, "float32")
			case tuple.TypeBOOL:
				val.ValBool = rapid.Bool().Draw(t, "bool")
			case tuple.TypeVARCHAR:
				val.ValStr = rapid.String().Draw(t, "string")
			case tuple.TypeVECTOR:
				dims := rapid.IntRange(1, 10).Draw(t, "dims")
				vec := make([]float32, dims)
				for j := 0; j < dims; j++ {
					vec[j] = rapid.Float32().Draw(t, "vecVal")
				}
				val.ValVec = vec
			}
		}
		vals[i] = val
	}
	return &tuple.Tuple{Schema: schema, Values: vals}
}

func compareValues(t *rapid.T, v1, v2 tuple.Value) {
	if v1.Type != v2.Type {
		t.Fatalf("type mismatch: %v != %v", v1.Type, v2.Type)
	}
	if v1.IsNull != v2.IsNull {
		t.Fatalf("isNull mismatch: %v != %v", v1.IsNull, v2.IsNull)
	}
	if v1.IsNull {
		return
	}
	switch v1.Type {
	case tuple.TypeINT32:
		if v1.ValInt32 != v2.ValInt32 {
			t.Fatalf("val mismatch: %v != %v", v1.ValInt32, v2.ValInt32)
		}
	case tuple.TypeINT64:
		if v1.ValInt64 != v2.ValInt64 {
			t.Fatalf("val mismatch: %v != %v", v1.ValInt64, v2.ValInt64)
		}
	case tuple.TypeFLOAT32:
		if math.IsNaN(float64(v1.ValFloat)) && math.IsNaN(float64(v2.ValFloat)) {
			return
		}
		if v1.ValFloat != v2.ValFloat {
			t.Fatalf("val mismatch: %v != %v", v1.ValFloat, v2.ValFloat)
		}
	case tuple.TypeBOOL:
		if v1.ValBool != v2.ValBool {
			t.Fatalf("val mismatch: %v != %v", v1.ValBool, v2.ValBool)
		}
	case tuple.TypeVARCHAR:
		if v1.ValStr != v2.ValStr {
			t.Fatalf("val mismatch: %v != %v", v1.ValStr, v2.ValStr)
		}
	case tuple.TypeVECTOR:
		if len(v1.ValVec) != len(v2.ValVec) {
			t.Fatalf("vec len mismatch: %v != %v", len(v1.ValVec), len(v2.ValVec))
		}
		for i := range v1.ValVec {
			if math.IsNaN(float64(v1.ValVec[i])) && math.IsNaN(float64(v2.ValVec[i])) {
				continue
			}
			if v1.ValVec[i] != v2.ValVec[i] {
				t.Fatalf("vec val mismatch at %d: %v != %v", i, v1.ValVec[i], v2.ValVec[i])
			}
		}
	}
}

func TestTupleRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		schema := genSchema(t)
		original := genTupleForSchema(t, schema)

		data, err := original.Serialize()
		if err != nil {
			t.Fatalf("failed to serialize: %v", err)
		}

		decoded, err := tuple.Deserialize(data, schema)
		if err != nil {
			t.Fatalf("failed to deserialize: %v", err)
		}

		if len(original.Values) != len(decoded.Values) {
			t.Fatalf("values len mismatch: %d != %d", len(original.Values), len(decoded.Values))
		}

		for i := range original.Values {
			compareValues(t, original.Values[i], decoded.Values[i])
		}
	})
}
