package llm

import (
	"strings"
	"testing"
)

func TestSplitForImport_SmallStaysWhole(t *testing.T) {
	in := "hej = hello\ntack = thanks"
	out := splitForImport(in, 4000)
	if len(out) != 1 || out[0] != in {
		t.Fatalf("small input should be one chunk unchanged; got %d chunks", len(out))
	}
}

func TestSplitForImport_SplitsOnLineBoundaries(t *testing.T) {
	// 10 lines of ~20 chars each; maxChars=50 forces several chunks.
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, strings.Repeat("a", 18))
	}
	in := strings.Join(lines, "\n")
	out := splitForImport(in, 50)
	if len(out) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(out))
	}
	// No chunk exceeds maxChars unless it's a single oversized line.
	for i, c := range out {
		if len(c) > 50 && strings.Count(c, "\n") > 0 {
			t.Errorf("chunk %d exceeds maxChars with multiple lines: %d", i, len(c))
		}
	}
	// Rejoining all chunks must reproduce the original exactly.
	if strings.Join(out, "") != in {
		t.Errorf("chunks do not reconstruct original input")
	}
}

func TestSplitForImport_OversizedSingleLine(t *testing.T) {
	in := strings.Repeat("x", 5000)
	out := splitForImport(in, 4000)
	if len(out) != 1 || out[0] != in {
		t.Fatalf("a single oversized line should stay intact in one chunk; got %d chunks", len(out))
	}
}

func TestSplitForImport_Reconstructs(t *testing.T) {
	in := "line1\nline2\nline3\nline4\n"
	out := splitForImport(in, 12)
	if strings.Join(out, "") != in {
		t.Errorf("reconstruction mismatch: %q", strings.Join(out, ""))
	}
}
