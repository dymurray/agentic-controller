package goose

import (
	"bytes"
	"strings"
	"testing"
)

func TestCaptureWriterBoundedTail(t *testing.T) {
	var live bytes.Buffer
	c := &captureWriter{w: &live}

	// Write well past the cap so truncation must occur, then a distinctive tail.
	head := bytes.Repeat([]byte("A"), 2*maxCapturedBytes+4096)
	tail := []byte("DISTINCTIVE_TAIL_END")
	if _, err := c.Write(head); err != nil {
		t.Fatalf("write head: %v", err)
	}
	if _, err := c.Write(tail); err != nil {
		t.Fatalf("write tail: %v", err)
	}

	// Live stream must receive every byte, unmodified.
	if live.Len() != len(head)+len(tail) {
		t.Errorf("live stream got %d bytes, want %d", live.Len(), len(head)+len(tail))
	}

	snap := c.snapshot()
	// Retained copy must be bounded, not the whole input.
	if len(snap) >= len(head)+len(tail) {
		t.Errorf("snapshot not bounded: %d bytes for %d input", len(snap), len(head)+len(tail))
	}
	if len(snap) > len(head) { // sanity: comfortably under total
		t.Errorf("snapshot larger than expected: %d bytes", len(snap))
	}
	s := string(snap)
	if !strings.Contains(s, "truncated") {
		t.Errorf("snapshot missing truncation marker")
	}
	if !strings.HasSuffix(s, "DISTINCTIVE_TAIL_END") {
		t.Errorf("snapshot dropped the most recent output")
	}
}

func TestCaptureWriterNoTruncation(t *testing.T) {
	var live bytes.Buffer
	c := &captureWriter{w: &live}
	in := []byte("short goose output\n")
	if _, err := c.Write(in); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := string(c.snapshot()); got != string(in) {
		t.Errorf("snapshot = %q, want %q", got, in)
	}
	if strings.Contains(string(c.snapshot()), "truncated") {
		t.Errorf("unexpected truncation marker for small output")
	}
}
