package utils

import (
	"errors"
	"testing"
)

// Since logger uses pterm which writes to stdout/stderr,
// these tests primarily ensure no panics occur during formatting.
func TestLogger(t *testing.T) {
	Success("success %s", "test")
	Info("info %s", "test")
	Warn("warn %s", "test")
	Error("error %s", "test")
	Debug("debug %s", "test")
	Box("box %s", "test")
}

func TestFatal(t *testing.T) {
	// We just ensure calling Fatal doesn't panic
	axErr := NewAxenError("test msg", "CODE")
	Fatal(axErr)

	stdErr := errors.New("standard error")
	Fatal(stdErr)
}
