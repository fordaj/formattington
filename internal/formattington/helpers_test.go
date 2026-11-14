package formattington

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeExecutableScript records every argument (one per line) in the file
// named by FAKE_ARGV_FILE and marks the invocation in FAKE_CALLED_FILE.
const fakeExecutableScript = "#!/bin/sh\n" +
	": > \"${FAKE_CALLED_FILE:-/dev/null}\"\n" +
	"printf '%s\\n' \"$@\" >> \"${FAKE_ARGV_FILE:-/dev/null}\"\n" +
	"exit 0\n"

// writeFakeExecutable creates a fake executable named name in dir.
func writeFakeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(fakeExecutableScript), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// fakeTools installs fake executables on PATH and isolates the bundled
// tools directory. It returns the directory holding the fakes, the file
// the fakes record their argv in, and the file they mark their
// invocations in.
func fakeTools(t *testing.T, names ...string) (binDir, argvFile, calledFile string) {
	t.Helper()
	root := t.TempDir()
	binDir = filepath.Join(root, "bin")
	argvFile = filepath.Join(root, "argv.txt")
	calledFile = filepath.Join(root, "called")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	if err := os.WriteFile(argvFile, nil, 0o644); err != nil {
		t.Fatalf("write %s: %v", argvFile, err)
	}

	emptyTools := filepath.Join(root, "empty-tools")
	if err := os.MkdirAll(emptyTools, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", emptyTools, err)
	}

	t.Setenv("FORMATTINGTON_TOOLS_DIR", emptyTools)
	t.Setenv("FORMATTINGTON_VERBOSE", "")
	t.Setenv("FAKE_ARGV_FILE", argvFile)
	t.Setenv("FAKE_CALLED_FILE", calledFile)
	t.Setenv("PATH", binDir)

	for _, name := range names {
		writeFakeExecutable(t, binDir, name)
	}
	return binDir, argvFile, calledFile
}

// toolWasCalled reports whether a fake executable recorded an invocation.
func toolWasCalled(t *testing.T, calledFile string) bool {
	t.Helper()
	_, err := os.Stat(calledFile)
	return err == nil
}

// recordedArgv reads the arguments recorded by the fake executables.
func recordedArgv(t *testing.T, argvFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("read %s: %v", argvFile, err)
	}
	text := string(data)
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	// Drop the empty element produced by the trailing newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

// captureOutput runs fn with stdout and stderr redirected to pipes and
// returns the exit code plus everything that was written.
func captureOutput(t *testing.T, fn func() int) (code int, stdout, stderr string) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	errRead, errWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout, os.Stderr = outWrite, errWrite

	outDone := make(chan string, 1)
	errDone := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(outRead)
		outDone <- string(data)
	}()
	go func() {
		data, _ := io.ReadAll(errRead)
		errDone <- string(data)
	}()

	code = fn()

	outWrite.Close()
	errWrite.Close()
	os.Stdout, os.Stderr = origOut, origErr

	return code, <-outDone, <-errDone
}

// runCLI runs the command line and returns the exit code and captured
// output.
func runCLI(t *testing.T, argv ...string) (code int, stdout, stderr string) {
	t.Helper()
	return captureOutput(t, func() int {
		return Main(argv)
	})
}

// requirePOSIX skips a test that needs POSIX shell fakes.
func requirePOSIX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fakes require POSIX")
	}
}

// writeFile writes content to path, creating parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// chmodExecutable marks path as executable.
func chmodExecutable(path string) error {
	return os.Chmod(path, 0o755)
}

// timeoutAfter returns a channel that fires after a generous delay, so a
// test that waits on it fails instead of hanging forever.
func timeoutAfter() <-chan time.Time {
	return time.After(30 * time.Second)
}
