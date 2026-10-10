package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchReleasePagesFindsOlderIroncladRelease(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("per_page = %q, want 100", got)
		}

		switch r.URL.Query().Get("page") {
		case "1":
			page := make([]githubRelease, releasesPerPage)
			for i := range page {
				page[i] = githubRelease{
					TagName: fmt.Sprintf("v3.0.%d", releasesPerPage-i),
					Body:    "<!-- pc-multitool-importance: normal -->",
				}
			}
			_ = json.NewEncoder(w).Encode(page)
		case "2":
			_ = json.NewEncoder(w).Encode([]githubRelease{{
				TagName: "v1.0.0",
				Body:    "<!-- pc-multitool-importance: important -->",
			}})
		default:
			_ = json.NewEncoder(w).Encode([]githubRelease{})
		}
	}))
	defer server.Close()

	client := server.Client()
	releases, err := fetchReleasePages(client, server.URL+"/releases?per_page=100")
	if err != nil {
		t.Fatalf("fetchReleasePages returned error: %v", err)
	}
	if len(releases) != releasesPerPage+1 {
		t.Fatalf("got %d releases, want %d", len(releases), releasesPerPage+1)
	}
	if calls != 2 {
		t.Fatalf("made %d page requests, want 2", calls)
	}

	got, found := selectUpdateRelease(releases, updateChannelIronclad, "v0.9.0")
	if !found {
		t.Fatal("expected Ironclad to find the important release on page 2")
	}
	if got.TagName != "v1.0.0" {
		t.Fatalf("selected %q, want v1.0.0", got.TagName)
	}
}
