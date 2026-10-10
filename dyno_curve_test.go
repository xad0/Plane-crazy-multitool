package main

import "testing"

func TestDynoPCHIPStaysInsideSegment(t *testing.T) {

	points := []dynoGraphPoint{
		{RPM: 0, BHP: 0},
		{RPM: 100, BHP: 12},
		{RPM: 200, BHP: 4},
		{RPM: 300, BHP: 20},
	}

	slopes := dynoPCHIPSlopes(points)

	if slopes == nil {
		t.Fatal("expected valid PCHIP slopes")
	}

	for i := 0; i < len(points)-1; i++ {

		low := points[i].BHP
		high := points[i].BHP

		if points[i+1].BHP < low {
			low = points[i+1].BHP
		}

		if points[i+1].BHP > high {
			high = points[i+1].BHP
		}

		for sample := 0; sample <= 100; sample++ {

			tValue :=
				float64(sample) /
					100

			value :=
				dynoPCHIPValue(
					points[i],
					points[i+1],
					slopes[i],
					slopes[i+1],
					tValue,
				)

			if value < low-1e-9 ||
				value > high+1e-9 {

				t.Fatalf(
					"segment %d produced %.6f outside [%.6f, %.6f]",
					i,
					value,
					low,
					high,
				)
			}
		}
	}
}

func TestDynoPCHIPRejectsDuplicateRPM(t *testing.T) {

	points := []dynoGraphPoint{
		{RPM: 0, BHP: 0},
		{RPM: 100, BHP: 10},
		{RPM: 100, BHP: 12},
	}

	if dynoPCHIPSlopes(points) != nil {
		t.Fatal("expected duplicate RPM values to reject interpolation")
	}
}

func TestDynoPCHIPHitsEndpoints(t *testing.T) {

	points := []dynoGraphPoint{
		{RPM: 0, BHP: 0},
		{RPM: 100, BHP: 10},
	}

	slopes := dynoPCHIPSlopes(points)

	if slopes == nil {
		t.Fatal("expected valid slopes")
	}

	if got := dynoPCHIPValue(
		points[0],
		points[1],
		slopes[0],
		slopes[1],
		0,
	); got != 0 {
		t.Fatalf("start = %.6f, want 0", got)
	}

	if got := dynoPCHIPValue(
		points[0],
		points[1],
		slopes[0],
		slopes[1],
		1,
	); got != 10 {
		t.Fatalf("end = %.6f, want 10", got)
	}
}
