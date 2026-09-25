package main

import (
	"math"
	"testing"
)

// TestCalculatePistonLength tests the core piston length formula:
//
//	length = (value × distance) / pistons
//
// Variables:
//   - distance: crank throw distance (Plane Crazy studs, any non-negative float)
//   - pistons:  number of pistons (positive integer)
//   - value:    coefficient / compressor value (non-negative float)
//   - result:   piston rod length to set in-game
func TestCalculatePistonLength(t *testing.T) {
	tests := []struct {
		distance float64
		pistons  int
		value    float64
		want     float64
	}{
		// Simple whole-number cases — result must be obvious.
		{1, 1, 1, 1},         // trivial identity
		{2, 2, 3.106, 3.106}, // web app golden value
		{4, 2, 2, 4},         // (2 × 4) / 2 = 4
		{3, 3, 3, 3},         // (3 × 3) / 3 = 3
		{10, 5, 2, 4},        // (2 × 10) / 5 = 4
		{1, 2, 3.106, 1.553}, // coefficient divided across two pistons

		// Zero distance always yields zero.
		{0, 2, 3.106, 0},
		{0, 1, 1, 0},

		// Zero value always yields zero.
		{5, 2, 0, 0},
	}

	const tolerance = 1e-9

	for _, tt := range tests {
		got := calculatePistonLength(tt.distance, tt.pistons, tt.value)

		if math.Abs(got-tt.want) > tolerance {
			t.Errorf(
				"calculatePistonLength(%v, %d, %v) = %v, want %v",
				tt.distance, tt.pistons, tt.value,
				got, tt.want,
			)
		}
	}
}

// TestCalculatePistonLengthInvalidInputs tests that invalid inputs return 0.
func TestCalculatePistonLengthInvalidInputs(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		pistons  int
		value    float64
	}{
		{"negative distance", -1, 2, 3.0},
		{"zero pistons", 2, 0, 3.0},
		{"negative pistons", 2, -1, 3.0},
		{"negative value", 2, 2, -1.0},
		{"NaN distance", math.NaN(), 2, 3.0},
		{"Inf distance", math.Inf(1), 2, 3.0},
		{"NaN value", 2, 2, math.NaN()},
		{"Inf value", 2, 2, math.Inf(1)},
	}

	for _, tt := range tests {
		got := calculatePistonLength(tt.distance, tt.pistons, tt.value)

		if got != 0 {
			t.Errorf(
				"%s: calculatePistonLength(%v, %d, %v) = %v, want 0",
				tt.name, tt.distance, tt.pistons, tt.value, got,
			)
		}
	}
}

// TestPistonLengthValid tests the validation predicate separately.
func TestPistonLengthValid(t *testing.T) {
	tests := []struct {
		distance float64
		pistons  int
		value    float64
		want     bool
	}{
		{1, 1, 1, true},
		{0, 1, 0, true},   // zero distance and value are allowed
		{-1, 1, 1, false}, // negative distance
		{1, 0, 1, false},  // zero pistons
		{1, -1, 1, false}, // negative pistons
		{1, 1, -1, false}, // negative value
		{math.NaN(), 1, 1, false},
		{math.Inf(1), 1, 1, false},
		{1, 1, math.NaN(), false},
		{1, 1, math.Inf(1), false},
	}

	for _, tt := range tests {
		got := pistonLengthValid(tt.distance, tt.pistons, tt.value)

		if got != tt.want {
			t.Errorf(
				"pistonLengthValid(%v, %d, %v) = %v, want %v",
				tt.distance, tt.pistons, tt.value, got, tt.want,
			)
		}
	}
}
