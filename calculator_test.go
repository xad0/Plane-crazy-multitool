package main

import (
	"math"
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		teeth      int
		wantFull   float64
		wantHalf   float64
		wantOffset float64
	}{
		// Minimum valid: 3 teeth. sin(360°/3) = sin(120°) ≠ 0, so no singularity.
		{3, 120.0, 60.0, -0.4226497308},
		// 4 and 5 are confirmed valid in Plane Crazy.
		{4, 90.0, 45.0, -0.2928932188},
		{5, 72.0, 36.0, -0.149349191648},
		{6, 60.0, 30.0, 0.0},
		{16, 22.5, 11.25, 1.5629154477},
		{20, 18.0, 9.0, 2.196226610750},
		{50, 7.2, 3.6, 6.962985554954},
		{100, 3.6, 1.8, 14.918112604549},
	}

	const tolerance = 1e-9

	for _, tt := range tests {
		full, half, offset := Calculate(tt.teeth)

		if math.Abs(full-tt.wantFull) > tolerance {
			t.Errorf(
				"Calculate(%d) full = %.12f, want %.12f",
				tt.teeth,
				full,
				tt.wantFull,
			)
		}

		if math.Abs(half-tt.wantHalf) > tolerance {
			t.Errorf(
				"Calculate(%d) half = %.12f, want %.12f",
				tt.teeth,
				half,
				tt.wantHalf,
			)
		}

		if math.Abs(offset-tt.wantOffset) > tolerance {
			t.Errorf(
				"Calculate(%d) offset = %.12f, want %.12f",
				tt.teeth,
				offset,
				tt.wantOffset,
			)
		}
	}
}

// TestCalculateInvalidBoundary verifies that teeth < 3 (singular values) return
// (0, 0, 0) to prevent divide-by-zero propagation.
func TestCalculateInvalidBoundary(t *testing.T) {
	invalidTeeth := []int{0, 1, 2, -5}

	for _, teeth := range invalidTeeth {
		full, half, offset := Calculate(teeth)

		if full != 0 || half != 0 || offset != 0 {
			t.Errorf(
				"Calculate(%d) = (%f, %f, %f), want (0, 0, 0)",
				teeth, full, half, offset,
			)
		}
	}
}

func TestAutoCompressors(t *testing.T) {
	tests := []struct {
		offset float64
		want   int
	}{
		{-0.293, 1},
		{0.0, 1},
		{0.5, 1},
		{1.0, 1},
		{1.0001, 2},
		{1.562915, 2},
		{2.1, 3},
		{7.4, 8},
	}

	for _, tt := range tests {
		got := AutoCompressors(tt.offset)

		if got != tt.want {
			t.Errorf(
				"AutoCompressors(%v) = %d, want %d",
				tt.offset,
				got,
				tt.want,
			)
		}
	}
}

func TestCompressorValue(t *testing.T) {
	tests := []struct {
		offset      float64
		compressors int
		want        float64
	}{
		{-0.293, 1, -0.293},
		{0.0, 1, 0.0},
		{0.563, 2, -0.437},
		{1.5629154477, 2, 0.5629154477},
		{1.5629154477, 1, 1.5629154477},
		{1.5629154477, 3, -0.4370845523},
	}

	const tolerance = 1e-9

	for _, tt := range tests {
		got := CompressorValue(tt.offset, tt.compressors)

		if math.Abs(got-tt.want) > tolerance {
			t.Errorf(
				"CompressorValue(%v, %d) = %.12f, want %.12f",
				tt.offset,
				tt.compressors,
				got,
				tt.want,
			)
		}
	}
}

func TestRound3(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0.0, 0.0},
		{0.5629154477, 0.563},
		{1.5629154477, 1.563},
		{-0.2928932188, -0.293},
		{1.2344, 1.234},
		{1.2345, 1.235},
	}

	for _, tt := range tests {
		got := Round3(tt.input)

		if got != tt.want {
			t.Errorf(
				"Round3(%v) = %v, want %v",
				tt.input,
				got,
				tt.want,
			)
		}
	}
}
