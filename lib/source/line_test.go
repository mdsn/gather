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

func TestLineBuffer_MultipleAdd(t *testing.T) {
	lb := NewLineBuffer(5)
	lb.Add([]byte("ron\nhermione\n"))

	lines := Collect(lb)
	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}

	if string(lines[0]) != "ron" || string(lines[1]) != "hermi" {
		t.Fatalf("wrong lines: '%s' '%s'", string(lines[0]), string(lines[1]))
	}

	lb.Add([]byte("harry\nhagrid"))

	lines = Collect(lb)
	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}

	if string(lines[0]) != "harry" || string(lines[1]) != "hagri" {
		t.Fatalf("wrong lines: '%s' '%s'", string(lines[0]), string(lines[1]))
	}
}

func TestLineBuffer_TruncatesAcrossAdd(t *testing.T) {
	lb := NewLineBuffer(5)
	// No ending newline; 'hermione' gets truncated and continues
	// truncating 'harry' below.
	lb.Add([]byte("ron\nhermione"))

	lines := Collect(lb)
	if len(lines) != 2 {
		t.Fatal("wrong length, want 2, got", len(lines))
	}

	if string(lines[0]) != "ron" || string(lines[1]) != "hermi" {
		t.Fatalf("wrong lines: '%s' '%s'", string(lines[0]), string(lines[1]))
	}

	lb.Add([]byte("harry\nhagrid"))

	lines = Collect(lb)
	if len(lines) != 1 {
		t.Fatal("wrong length, want 1, got", len(lines))
	}

	if string(lines[0]) != "hagri" {
		t.Fatal("wrong line:", string(lines[0]))
	}
}

func TestLineBuffer_AcrossAdd(t *testing.T) {
	lb := NewLineBuffer(40)

	lb.Add([]byte("Thou pimp "))
	lines := Collect(lb)
	if len(lines) != 0 {
		t.Fatal("wrong length, want 0, got", len(lines))
	}

	lb.Add([]byte("most infamous, "))
	lines = Collect(lb)
	if len(lines) != 0 {
		t.Fatal("wrong length, want 0, got", len(lines))
	}

	lb.Add([]byte("be still!\n"))
	lines = Collect(lb)
	if len(lines) != 1 {
		t.Fatal("wrong length, want 1, got", len(lines))
	}

	want := "Thou pimp most infamous, be still!"
	if string(lines[0]) != want {
		t.Fatal("wrong string, got:", string(lines[0]))
	}
}
