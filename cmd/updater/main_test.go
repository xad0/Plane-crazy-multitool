package main

import "testing"

func TestParseUpdaterArgsSupportsOldAndNewProtocols(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantVersion string
		wantPID     int
		wantErr     bool
	}{
		{
			name:        "original three argument protocol",
			args:        []string{"update.zip", "app", "123"},
			wantVersion: "",
			wantPID:     123,
		},
		{
			name:        "optional version protocol",
			args:        []string{"update.zip", "app", "123", "v2.3.0"},
			wantVersion: "v2.3.0",
			wantPID:     123,
		},
		{
			name:    "invalid pid",
			args:    []string{"update.zip", "app", "pid"},
			wantErr: true,
		},
		{
			name:    "wrong argument count",
			args:    []string{"update.zip", "app"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUpdaterArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.version != tt.wantVersion || got.parentPID != tt.wantPID {
				t.Fatalf("got version %q PID %d, want version %q PID %d", got.version, got.parentPID, tt.wantVersion, tt.wantPID)
			}
			if got.newPath != tt.args[0] || got.targetPath != tt.args[1] {
				t.Fatalf("got paths %q and %q, want %q and %q", got.newPath, got.targetPath, tt.args[0], tt.args[1])
			}
		})
	}
}
