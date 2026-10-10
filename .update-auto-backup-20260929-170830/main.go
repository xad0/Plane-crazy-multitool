package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const (
	maxWait    = 15 * time.Second
	retryDelay = 250 * time.Millisecond
)

func main() {
	if len(os.Args) != 4 {
		fail("usage: updater <new-file> <target> <parent-pid>")
	}

	newPath := os.Args[1]
	targetPath := os.Args[2]
	parentPID, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fail(fmt.Sprintf("invalid parent PID: %v", err))
	}

	// Give the main application time to terminate.
	time.Sleep(750 * time.Millisecond)

	// The updater must never wait forever.
	deadline := time.Now().Add(maxWait)

	for {
		if !processIsRunning(parentPID) {
			break
		}

		if time.Now().After(deadline) {
			fail("timed out waiting for application to exit")
		}

		time.Sleep(retryDelay)
	}

	if err := installUpdate(newPath, targetPath); err != nil {
		fail(err.Error())
	}

	cleanup(newPath)
}

func processIsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(
			"tasklist",
			"/FI",
			fmt.Sprintf("PID eq %d", pid),
			"/NH",
		)

		output, err := cmd.Output()
		if err != nil {
			return false
		}

		return len(output) > 0 &&
			containsPID(string(output), pid)

	default:
		err := exec.Command(
			"kill",
			"-0",
			strconv.Itoa(pid),
		).Run()

		return err == nil
	}
}

func containsPID(output string, pid int) bool {
	needle := strconv.Itoa(pid)

	for _, line := range splitLines(output) {
		fields := splitFields(line)
		if len(fields) >= 2 && fields[1] == needle {
			return true
		}
	}

	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

func splitFields(s string) []string {
	var fields []string
	start := -1

	for i := 0; i < len(s); i++ {
		isSpace := s[i] == ' ' || s[i] == '\t' || s[i] == '\r'
		if !isSpace && start == -1 {
			start = i
		}

		if isSpace && start != -1 {
			fields = append(fields, s[start:i])
			start = -1
		}
	}

	if start != -1 {
		fields = append(fields, s[start:])
	}

	return fields
}

func installUpdate(zipPath, targetPath string) error {
	targetDir := filepath.Dir(targetPath)

	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("cannot stat current application: %w", err)
	}

	stagedPath := filepath.Join(
		targetDir,
		".pc-gear-calculator-update.tmp",
	)

	backupPath := filepath.Join(
		targetDir,
		".pc-gear-calculator-backup",
	)

	_ = os.Remove(stagedPath)

	if err := extractApplication(
		zipPath,
		filepath.Base(targetPath),
		stagedPath,
	); err != nil {
		return fmt.Errorf("cannot extract update: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(
			stagedPath,
			info.Mode().Perm(),
		); err != nil {
			os.Remove(stagedPath)
			return fmt.Errorf(
				"cannot set update permissions: %w",
				err,
			)
		}
	}

	_ = os.Remove(backupPath)

	if err := os.Rename(targetPath, backupPath); err != nil {
		os.Remove(stagedPath)

		return fmt.Errorf(
			"cannot back up current application: %w",
			err,
		)
	}

	if err := os.Rename(stagedPath, targetPath); err != nil {
		_ = os.Rename(backupPath, targetPath)
		os.Remove(stagedPath)

		return fmt.Errorf(
			"cannot install new application: %w",
			err,
		)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(
			targetPath,
			info.Mode().Perm(),
		); err != nil {
			_ = os.Remove(targetPath)
			_ = os.Rename(backupPath, targetPath)

			return fmt.Errorf(
				"cannot set installed permissions: %w",
				err,
			)
		}
	}

	if err := startUpdatedApp(targetPath); err != nil {
		_ = os.Remove(targetPath)
		_ = os.Rename(backupPath, targetPath)

		return fmt.Errorf(
			"cannot start updated application: %w",
			err,
		)
	}

	_ = os.Remove(backupPath)

	return nil
}

func extractApplication(
	zipPath string,
	appName string,
	destination string,
) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	expected := "PC-Multitool/" + filepath.ToSlash(appName)

	for _, entry := range reader.File {
		name := filepath.ToSlash(entry.Name)

		if name != expected {
			continue
		}

		if entry.FileInfo().IsDir() {
			return fmt.Errorf(
				"application entry is a directory",
			)
		}

		in, err := entry.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(destination)
		if err != nil {
			_ = in.Close()
			return err
		}

		_, copyErr := io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()

		if copyErr != nil {
			os.Remove(destination)
			return copyErr
		}

		if closeOutErr != nil {
			os.Remove(destination)
			return closeOutErr
		}

		if closeInErr != nil {
			os.Remove(destination)
			return closeInErr
		}

		return nil
	}

	return fmt.Errorf(
		"ZIP does not contain %s",
		expected,
	)
}

func startUpdatedApp(targetPath string) error {
	cmd := exec.Command(targetPath)
	cmd.Dir = filepath.Dir(targetPath)

	return cmd.Start()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}

	return nil
}

func cleanup(newPath string) {
	_ = os.Remove(newPath)

	if helper, err := os.Executable(); err == nil {
		// Delayed self-delete.
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = os.Remove(helper)
		}()
	}
}

func fail(message string) {
	errorPath := filepath.Join(
		os.TempDir(),
		"pc-gear-calculator-updater-error",
	)

	_ = os.WriteFile(
		errorPath,
		[]byte(message),
		0600,
	)

	os.Exit(1)
}
