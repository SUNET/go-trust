package main

import (
	"bytes"
	"os"
	"testing"
)

func TestUsageOutput(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	usage()
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()
	if len(output) == 0 || output[:5] != "\nUsag" {
		t.Errorf("usage() did not print expected usage message, got: %q", output)
	}
}
