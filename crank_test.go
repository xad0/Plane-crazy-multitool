package main

import "testing"

func TestCrankValid(t *testing.T) {
	tests := []struct {
		layout    crankLayout
		cylinders int
		want      bool
	}{
		{crankInline, 1, true},
		{crankInline, 4, true},
		{crankInline, 6, true},
		{crankInline, 0, false},
		{crankInline, -1, false},

		{crankV, 4, true},
		{crankV, 5, true},
		{crankV, 6, true},
		{crankV, 7, true},
		{crankV, 0, false},

		{crankBoxer, 4, true},
		{crankBoxer, 5, true},
		{crankBoxer, 6, true},
		{crankBoxer, 0, false},
	}

	for _, tt := range tests {
		got := crankValid(tt.layout, tt.cylinders)
		if got != tt.want {
			t.Errorf(
				"crankValid(%v, %d) = %v, want %v",
				tt.layout, tt.cylinders, got, tt.want,
			)
		}
	}
}

func TestCrankEffectivePositions(t *testing.T) {
	tests := []struct {
		layout    crankLayout
		cylinders int
		want      int
	}{
		{crankInline, 1, 1},
		{crankInline, 4, 4},
		{crankInline, 6, 6},
		{crankInline, 0, 0},

		{crankV, 4, 2},
		{crankV, 5, 3},
		{crankV, 6, 3},
		{crankV, 7, 4},
		{crankV, 0, 0},

		{crankBoxer, 5, 3},
		{crankBoxer, 6, 3},
	}

	for _, tt := range tests {
		got := crankEffectivePositions(tt.layout, tt.cylinders)
		if got != tt.want {
			t.Errorf(
				"crankEffectivePositions(%v, %d) = %d, want %d",
				tt.layout, tt.cylinders, got, tt.want,
			)
		}
	}
}

func TestCrankAngle(t *testing.T) {
	tests := []struct {
		layout    crankLayout
		cylinders int
		want      float64
	}{
		{crankInline, 4, 90.0},
		{crankInline, 6, 60.0},

		{crankV, 4, 180.0},
		{crankV, 5, 120.0},
		{crankV, 6, 120.0},
		{crankV, 7, 90.0},

		{crankBoxer, 5, 120.0},
		{crankBoxer, 6, 120.0},

		{crankV, 0, 0.0},
		{crankInline, 0, 0.0},
	}

	for _, tt := range tests {
		got := crankAngle(tt.layout, tt.cylinders)
		if got != tt.want {
			t.Errorf(
				"crankAngle(%v, %d) = %f, want %f",
				tt.layout, tt.cylinders, got, tt.want,
			)
		}
	}
}
