package logger

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestParseLevel(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int8
	}{
		{"debug", -1},
		{"DEBUG", -1},
		{"info", 0},
		{"INFO", 0},
		{"warn", 1},
		{"warning", 1},
		{"error", 2},
		{"", 0},        // default
		{"bogus", 0},   // default
		{"  info ", 0}, // trimmed
	} {
		got := ParseLevel(tt.in)
		if int8(got) != tt.want {
			t.Errorf("ParseLevel(%q) = %d, want %d", tt.in, int8(got), tt.want)
		}
	}
}

func TestNew_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	log, err := New(dir, zap.InfoLevel)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Sync()

	log.Info("test message")
	log.Sync()

	logPath := filepath.Join(dir, "tork.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if len(data) == 0 {
		t.Error("log file is empty")
	}
}

func TestNew_NestedDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "deeply", "nested")
	log, err := New(dir, zap.DebugLevel)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Sync()

	log.Debug("debug message")
	log.Sync()

	if _, err := os.Stat(filepath.Join(dir, "tork.log")); err != nil {
		t.Fatal("log file not created in nested dir")
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	dir := t.TempDir()
	log, err := New(dir, zap.ErrorLevel)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Sync()

	log.Info("should be filtered")
	log.Warn("should be filtered")
	log.Sync()

	logPath := filepath.Join(dir, "tork.log")
	data, _ := os.ReadFile(logPath)
	if len(data) != 0 {
		t.Error("info/warn messages should not be logged at error level")
	}
}

func TestNop_DoesNotPanic(t *testing.T) {
	log := Nop()
	log.Debug("nop debug")
	log.Info("nop info")
	log.Warn("nop warn")
	log.Error("nop error")
	if err := log.Sync(); err != nil {
		t.Errorf("Nop().Sync() error: %v", err)
	}
}
