package source

import (
	"testing"
)

func Collect(lb *LineBuffer) [][]byte {
	lines := make([][]byte, 0)
	for line := range lb.Lines() {
		lines = append(lines, line)
	}
	return lines
}

func TestLineBuffer_YieldsOne(t *testing.T) {
	lb := NewLineBuffer(10)
	lb.Add([]byte("rudolph\n"))
	lines := Collect(lb)

	if len(lines) != 1 {
		t.Fatal("wrong length, want 1, got", len(lines))
	}

	if string(lines[0]) != "rudolph" {
		t.Fatalf("wrong line, want 'rudolph', got '%s'", string(lines[0]))
	}
}

func TestLineBuffer_YieldsTwo(t *testing.T) {
	lb := NewLineBuffer(20)
	lb.Add([]byte("polonious\nfortinbras\n"))
	lines := Collect(lb)

	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}

	if string(lines[0]) != "polonious" ||
		string(lines[1]) != "fortinbras" {
		t.Fatal("wrong lines:", string(lines[0]), string(lines[1]))
	}
}

func TestLineBuffer_Truncates(t *testing.T) {
	lb := NewLineBuffer(5)
	lb.Add([]byte("horatio"))
	lines := Collect(lb)

	if len(lines) != 1 {
		t.Fatal("wrong length, want 1, got", len(lines))
	}

	if string(lines[0]) != "horat" {
		t.Fatalf("wrong line, want 'horat', got '%s'", string(lines[0]))
	}
}

func TestLineBuffer_TruncatesThenYields(t *testing.T) {
	lb := NewLineBuffer(5)
	lb.Add([]byte("horatio\nhamlet"))
	lines := Collect(lb)

	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}

	if string(lines[0]) != "horat" || string(lines[1]) != "hamle" {
		t.Fatalf("wrong lines: '%s' '%s'", string(lines[0]), string(lines[1]))
	}
}
