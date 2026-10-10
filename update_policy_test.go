package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseImportance(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing marker defaults to normal",
			body: "normal release notes",
			want: releaseImportanceNormal,
		},
		{
			name: "normal",
			body: "<!-- pc-multitool-importance: normal -->",
			want: releaseImportanceNormal,
		},
		{
			name: "important",
			body: "<!-- pc-multitool-importance: important -->",
			want: releaseImportanceImportant,
		},
		{
			name: "critical with surrounding notes",
			body: "Some notes\n\n<!-- pc-multitool-importance: critical -->\n",
			want: releaseImportanceCritical,
		},
		{
			name: "invalid marker defaults to normal",
			body: "<!-- pc-multitool-importance: massive -->",
			want: releaseImportanceNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := releaseImportance(
				githubRelease{
					Body: tt.body,
				},
			)

			if got != tt.want {
				t.Fatalf(
					"got %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestReleaseAllowedForChannel(t *testing.T) {
	tests := []struct {
		name       string
		channel    string
		importance string
		want       bool
	}{
		{
			name:       "edge accepts normal",
			channel:    updateChannelEdge,
			importance: releaseImportanceNormal,
			want:       true,
		},
		{
			name:       "edge accepts important",
			channel:    updateChannelEdge,
			importance: releaseImportanceImportant,
			want:       true,
		},
		{
			name:       "workshop accepts release for manual checks",
			channel:    updateChannelWorkshop,
			importance: releaseImportanceNormal,
			want:       true,
		},
		{
			name:       "ironclad rejects normal",
			channel:    updateChannelIronclad,
			importance: releaseImportanceNormal,
			want:       false,
		},
		{
			name:       "ironclad accepts important",
			channel:    updateChannelIronclad,
			importance: releaseImportanceImportant,
			want:       true,
		},
		{
			name:       "ironclad accepts critical",
			channel:    updateChannelIronclad,
			importance: releaseImportanceCritical,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := githubRelease{
				Body: "<!-- pc-multitool-importance: " +
					tt.importance +
					" -->",
			}

			got :=
				releaseAllowedForChannel(
					release,
					tt.channel,
				)

			if got != tt.want {
				t.Fatalf(
					"got %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestSelectUpdateReleaseIroncladSkipsNormal(t *testing.T) {
	releases := []githubRelease{
		{
			TagName: "v1.2.0",
			Body:    "<!-- pc-multitool-importance: normal -->",
		},
		{
			TagName: "v1.1.0",
			Body:    "<!-- pc-multitool-importance: important -->",
		},
		{
			TagName: "v1.0.5",
			Body:    "<!-- pc-multitool-importance: critical -->",
		},
	}

	got, found :=
		selectUpdateRelease(
			releases,
			updateChannelIronclad,
			"v1.0.0",
		)

	if !found {
		t.Fatal("expected an eligible release")
	}

	if got.TagName != "v1.1.0" {
		t.Fatalf(
			"got %q, want v1.1.0",
			got.TagName,
		)
	}
}

func TestSelectUpdateReleaseEdgeGetsNewestRelease(t *testing.T) {
	releases := []githubRelease{
		{
			TagName: "v1.1.0",
			Body:    "<!-- pc-multitool-importance: important -->",
		},
		{
			TagName: "v1.2.0",
			Body:    "<!-- pc-multitool-importance: normal -->",
		},
		{
			TagName: "v1.0.9",
			Body:    "<!-- pc-multitool-importance: critical -->",
		},
	}

	got, found :=
		selectUpdateRelease(
			releases,
			updateChannelEdge,
			"v1.0.0",
		)

	if !found {
		t.Fatal("expected an eligible release")
	}

	if got.TagName != "v1.2.0" {
		t.Fatalf(
			"got %q, want v1.2.0",
			got.TagName,
		)
	}
}

func TestSelectLatestEligibleReleaseIroncladCanBeOlder(t *testing.T) {
	releases := []githubRelease{
		{
			TagName: "v1.2.0",
			Body:    "<!-- pc-multitool-importance: normal -->",
		},
		{
			TagName: "v1.1.0",
			Body:    "<!-- pc-multitool-importance: important -->",
		},
		{
			TagName: "v1.0.5",
			Body:    "<!-- pc-multitool-importance: critical -->",
		},
	}

	got, found :=
		selectLatestEligibleRelease(
			releases,
			updateChannelIronclad,
			"v1.2.0",
		)

	if !found {
		t.Fatal("expected an eligible Ironclad release")
	}

	if got.TagName != "v1.1.0" {
		t.Fatalf(
			"got %q, want v1.1.0",
			got.TagName,
		)
	}
}

func TestSelectUpdateReleaseStableSkipsPrerelease(t *testing.T) {
	releases := []githubRelease{
		{
			TagName:    "v1.3.0-rc.1",
			Prerelease: true,
		},
		{
			TagName: "v1.2.0",
		},
	}

	got, found :=
		selectUpdateRelease(
			releases,
			updateChannelEdge,
			"v1.0.0",
		)

	if !found {
		t.Fatal("expected an eligible release")
	}

	if got.TagName != "v1.2.0" {
		t.Fatalf(
			"got %q, want v1.2.0",
			got.TagName,
		)
	}
}

func TestNormalizeUpdateVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "v2.3.0", want: "2.3.0"},
		{input: "2.3.0", want: "2.3.0"},
		{input: " V2.3.0 ", want: "2.3.0"},
		{input: "dev", want: "dev"},
	}
	for _, tt := range tests {
		if got := normalizeUpdateVersion(tt.input); got != tt.want {
			t.Errorf("normalizeUpdateVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConsumeUpdateMarkerOnlyAcceptsInstalledVersion(t *testing.T) {
	tests := []struct {
		name           string
		markerVersion  string
		currentVersion string
		want           string
	}{
		{name: "matching tag and executable version", markerVersion: "v2.3.0", currentVersion: "2.3.0", want: "v2.3.0"},
		{name: "stale marker is ignored", markerVersion: "v2.3.0", currentVersion: "2.2.0", want: ""},
		{name: "empty marker is ignored", markerVersion: "", currentVersion: "2.3.0", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "update-marker")
			if err := os.WriteFile(path, []byte(tt.markerVersion), 0600); err != nil {
				t.Fatal(err)
			}
			if got := consumeUpdateMarker(path, tt.currentVersion); got != tt.want {
				t.Fatalf("consumeUpdateMarker() = %q, want %q", got, tt.want)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("marker should be removed after consumption, stat error = %v", err)
			}
		})
	}
}
