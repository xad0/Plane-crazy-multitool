package main

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestCopyMarkersHaveZeroVisibleWidthAndAreRemoved(t *testing.T) {
	c := &copyState{}
	display := "12.500°"
	marked := c.markValue("12.500", display)
	if got := lipgloss.Width(marked); got != lipgloss.Width(display) {
		t.Fatalf("marked width = %d, display width = %d", got, lipgloss.Width(display))
	}
	final := c.finalizeContent("label\n  "+marked+" tail", getPalette("catppuccin-mocha", "green"))
	if strings.ContainsAny(final, "\x00\x01\x02") {
		t.Fatalf("copy marker control bytes leaked into view content: %q", final)
	}
	if !strings.Contains(final, "label\n  "+display+" tail") {
		t.Fatalf("final content changed visible text: %q", final)
	}
	if got := len(c.targets); got != 1 {
		t.Fatalf("registered %d targets, want 1", got)
	}
	target := c.targets[0]
	if target.x1 != 2 || target.y != 1 || target.rawValue != "12.500" || target.x2-target.x1 != lipgloss.Width(display) {
		t.Fatalf("unexpected target: %#v", target)
	}
	if raw, ok := c.checkClick(target.x1, target.y); !ok || raw != "12.500" {
		t.Fatalf("checkClick() = %q, %v; want 12.500, true", raw, ok)
	}
}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

var copyTestANSISequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func TestCopyMarkerCanBeHovered(t *testing.T) {
	c := &copyState{hasHover: true, hoverX: 0, hoverY: 0}
	marked := c.markValue("90.000", "90.000°")
	final := c.finalizeContent(marked, getPalette("catppuccin-mocha", "green"))
	visible := copyTestANSISequence.ReplaceAllString(final, "")
	if !strings.Contains(visible, "90.000°") {
		t.Fatalf("hover rendering lost the value: visible=%q, raw=%q", visible, final)
	}
	if len(c.targets) != 1 {
		t.Fatalf("registered %d targets, want 1", len(c.targets))
	}
}

func TestCopyMarkersSurviveLipGlossStyleWrapping(t *testing.T) {
	c := &copyState{}
	marked := c.markValue("12.500", "12.500°")
	styled := lipgloss.NewStyle().Bold(true).Render(marked)
	final := c.finalizeContent(styled, getPalette("catppuccin-mocha", "green"))
	if len(c.targets) != 1 || c.targets[0].rawValue != "12.500" {
		t.Fatalf("styled marker was not registered: %#v; content=%q", c.targets, final)
	}
	if strings.ContainsAny(final, "\x00\x01\x02") {
		t.Fatalf("styled marker bytes leaked into final content: %q", final)
	}
}

func TestClearCopyTargetsResetsMarkerRegistryOnly(t *testing.T) {
	c := &copyState{toastText: "Copied 5", toastSuccess: true, hasHover: true, hoverX: 5, hoverY: 6}
	c.markValue("5", "5")
	c.addTarget(1, 2, 1, "old")
	c.clearTargets()
	if len(c.targets) != 0 || len(c.markedValues) != 0 || c.nextMarkID != 0 {
		t.Fatalf("copy marker registry was not cleared: %#v", c)
	}
	if c.toastText != "Copied 5" || !c.toastSuccess || !c.hasHover || c.hoverX != 5 || c.hoverY != 6 {
		t.Fatalf("clearTargets should preserve toast and hover state: %#v", c)
	}
}

func TestCopyMarkerEncodingAvoidsLayoutControlBytes(t *testing.T) {
	for _, token := range []string{copyMarkerToken(1, true), copyMarkerToken(65535, false)} {
		if strings.ContainsAny(token, "\x00\x01\x02\t\n\r\v\f\a") {
			t.Fatalf("marker contains a layout-changing control byte: %q", token)
		}
		if got := lipgloss.Width(token); got != 0 {
			t.Fatalf("marker width = %d, want zero: %q", got, token)
		}
	}
}

func TestReferenceTeethColumnIsNotCopyable(t *testing.T) {
	var copied []string
	mark := func(rawValue, displayText string) string {
		copied = append(copied, rawValue)
		return displayText
	}
	_ = buildReferenceRow(
		6, 60, 30, 1.234, 0.567, 2,
		[]refColumn{refTeeth, refFull, refHalf, refOffset, refCompValue, refCompressors},
		4, mark,
	)
	if len(copied) != 5 {
		t.Fatalf("copy callback called %d times, want 5 (calculated columns only)", len(copied))
	}
	for _, value := range copied {
		if value == "6" {
			t.Fatalf("TEETH value was registered as copyable: %q", value)
		}
	}
}

func TestHoveredCopyValueRestoresPanelSGRState(t *testing.T) {
	c := &copyState{hasHover: true, hoverX: 0, hoverY: 0}
	panelStyle := "\x1b[38;2;205;214;244;48;2;30;30;46m"
	marked := c.markValue("90.000", "90.000°")
	final := c.finalizeContent(panelStyle+marked+" remaining chart", getPalette("catppuccin-mocha", "green"))

	rest := strings.Index(final, " remaining chart")
	if rest < 0 {
		t.Fatalf("trailing chart text disappeared: %q", final)
	}
	restored := strings.LastIndex(final[:rest], panelStyle)
	if restored < 0 {
		t.Fatalf("panel SGR state was not restored after hovered value: %q", final)
	}
	if len(c.targets) != 1 || c.targets[0].rawValue != "90.000" || c.targets[0].x2-c.targets[0].x1 != lipgloss.Width("90.000°") {
		t.Fatalf("unexpected copy target: %#v", c.targets)
	}
}
