package main

import (
	"strings"
	"testing"
)

func TestDynoGraphSeriesModes(t *testing.T) {
	points := []dynoGraphPoint{
		{SPS: 0, RPM: 0, Torque: 0, BHP: 0},
		{SPS: 10, RPM: 38.4, Torque: 4, BHP: 2.9},
	}

	bhp := dynoGraphSeries(points, dynoGraphBHP)
	if bhp[1].RPM != 38.4 || bhp[1].BHP != 2.9 {
		t.Fatalf("BHP/RPM series changed unexpectedly: %#v", bhp[1])
	}
	torque := dynoGraphSeries(points, dynoGraphTorque)
	if torque[1].RPM != 10 || torque[1].BHP != 4 {
		t.Fatalf("Torque/SPS mapping is wrong: %#v", torque[1])
	}
	if got := dynoGraphModeTitle(dynoGraphTorque); got != "Torque vs SPS" {
		t.Fatalf("wrong torque mode title: %q", got)
	}
	if got := dynoGraphModeTitle(dynoGraphBHP); got != "BHP vs RPM" {
		t.Fatalf("wrong BHP mode title: %q", got)
	}
}

func TestDynoRemovePointKeepsOneRowMinimum(t *testing.T) {
	rows := []dynoPoint{newDynoPoint(), newDynoPoint(), newDynoPoint()}
	remaining, ok := dynoRemovePoint(rows, 1)
	if !ok || len(remaining) != 2 {
		t.Fatalf("remove row result = %d rows, removed=%v", len(remaining), ok)
	}
	if _, ok := dynoRemovePoint(remaining, -1); ok {
		t.Fatal("invalid index should not remove a row")
	}
	if _, ok := dynoRemovePoint(remaining[:1], 0); ok {
		t.Fatal("last row must not be removable")
	}
}

func TestDynoGraphTextIncludesSelectedMode(t *testing.T) {
	points := []dynoGraphPoint{{SPS: 0, RPM: 0, Torque: 0, BHP: 0}, {SPS: 10, RPM: 38.4, Torque: 4, BHP: 2.9}}
	p := getPalette("catppuccin-mocha", "green")
	view := dynoGraphText(points, 60, 18, p, dynoGraphTorque)
	if !strings.Contains(view, "Torque vs SPS") || !strings.Contains(view, "SPS →") || !strings.Contains(view, "Torque ↑") {
		t.Fatalf("torque graph labels missing:\n%s", view)
	}
}
