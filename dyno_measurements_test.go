package main

import "testing"

func TestDynoMeasuredPointsExcludesSyntheticOrigin(t *testing.T) {
	points := []dynoGraphPoint{
		{SPS: 0, RPM: 0, Torque: 0, BHP: 0},
		{SPS: 10, RPM: 38.4, Torque: 4, BHP: 2.9},
		{SPS: 20, RPM: 76.8, Torque: 6, BHP: 8.8},
	}

	got := dynoMeasuredPoints(points)
	if len(got) != 2 {
		t.Fatalf("got %d measurements, want 2", len(got))
	}
	if got[0].SPS != 10 || got[1].SPS != 20 {
		t.Fatalf("synthetic origin was not excluded: %#v", got)
	}
}

func TestDynoMeasuredPointsEmptyWithOnlyOrigin(t *testing.T) {
	if got := dynoMeasuredPoints([]dynoGraphPoint{{RPM: 0, BHP: 0}}); len(got) != 0 {
		t.Fatalf("got %d measurements, want 0", len(got))
	}
}
