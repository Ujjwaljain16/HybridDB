package serialization_test

import (
	"math"
	"testing"

	"github.com/Ujjwaljain16/hybriddb/internal/storage/serialization"
	"pgregory.net/rapid"
)

func TestUint32RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		val := rapid.Uint32().Draw(t, "val")
		buf := make([]byte, 4)
		serialization.WriteUint32(buf, val)
		out := serialization.ReadUint32(buf)
		if val != out {
			t.Fatalf("expected %d, got %d", val, out)
		}
	})
}

func TestFloat32RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		val := rapid.Float32().Draw(t, "val")
		buf := make([]byte, 4)
		serialization.WriteFloat32(buf, val)
		out := serialization.ReadFloat32(buf)
		
		// Handle NaN comparison
		if math.IsNaN(float64(val)) && math.IsNaN(float64(out)) {
			return
		}
		if val != out {
			t.Fatalf("expected %f, got %f", val, out)
		}
	})
}

func TestVectorRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dims := rapid.Uint32Range(1, 1000).Draw(t, "dims")
		vec := make([]float32, dims)
		for i := range vec {
			vec[i] = rapid.Float32().Draw(t, "val")
		}
		
		buf := make([]byte, dims*4)
		serialization.WriteVector(buf, vec)
		out := serialization.ReadVector(buf, dims)
		
		for i := range vec {
			// Handle NaN comparison
			if math.IsNaN(float64(vec[i])) && math.IsNaN(float64(out[i])) {
				continue
			}
			if vec[i] != out[i] {
				t.Fatalf("expected %f at index %d, got %f", vec[i], i, out[i])
			}
		}
	})
}

func TestChecksumDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := rapid.SliceOf(rapid.Byte()).Draw(t, "data")
		cs1 := serialization.ComputeChecksum(data)
		cs2 := serialization.ComputeChecksum(data)
		if cs1 != cs2 {
			t.Fatalf("checksum not deterministic: %d != %d", cs1, cs2)
		}
	})
}
