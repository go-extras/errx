package json

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateMessage_HardCutPath exercises the branch in truncateMessage where
// maxBytes is too small to fit the truncationSuffix. In that case the function
// must hard-cut at maxBytes but back off to a valid UTF-8 rune boundary so the
// returned string is never an invalid UTF-8 sequence.
//
// This locks in coverage for inputs that start with a multi-byte rune (e.g. "€"
// is 3 bytes in UTF-8) with very small byte budgets.
func TestTruncateMessage_HardCutPath(t *testing.T) {
	// "€" is 3 bytes in UTF-8 (E2 82 AC), so any non-zero cut < 3 must back off
	// all the way to 0 to remain valid UTF-8.
	const euro = "€..." // 3 + 3 = 6 bytes total

	tests := []struct {
		name     string
		input    string
		maxBytes int
		want     string
	}{
		{
			name:     "euro cut at 2 bytes returns empty",
			input:    euro,
			maxBytes: 2,
			want:     "",
		},
		{
			name:     "euro cut at 1 byte returns empty",
			input:    euro,
			maxBytes: 1,
			want:     "",
		},
		{
			name:     "euro cut at 0 bytes returns input unchanged (no-op branch)",
			input:    euro,
			maxBytes: 0,
			want:     euro,
		},
		{
			name:     "ascii cut at 3 bytes (smaller than suffix) hard-cuts",
			input:    "hello",
			maxBytes: 3,
			want:     "hel",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateMessage(tc.input, tc.maxBytes)

			if got != tc.want {
				t.Errorf("truncateMessage(%q, %d) = %q, want %q",
					tc.input, tc.maxBytes, got, tc.want)
			}

			// Lock in UTF-8 safety regardless of how the function evolves.
			if !utf8.ValidString(got) {
				t.Errorf("truncateMessage(%q, %d) returned invalid UTF-8: % x",
					tc.input, tc.maxBytes, got)
			}

			// The hard-cut branch must never exceed the byte budget (when active).
			if tc.maxBytes > 0 && len(got) > tc.maxBytes {
				t.Errorf("truncateMessage(%q, %d) returned %d bytes, exceeds limit",
					tc.input, tc.maxBytes, len(got))
			}
		})
	}
}

// TestTruncateMessage_HardCutPath_NoReplacementChar guards against a regression
// where a naive cut would emit U+FFFD when re-decoded.
func TestTruncateMessage_HardCutPath_NoReplacementChar(t *testing.T) {
	long := strings.Repeat("€", 10) // 30 bytes, all multi-byte runes
	for maxBytes := 1; maxBytes <= len(truncationSuffix); maxBytes++ {
		got := truncateMessage(long, maxBytes)
		if !utf8.ValidString(got) {
			t.Errorf("truncateMessage(maxBytes=%d) invalid UTF-8: % x", maxBytes, got)
		}
		if strings.ContainsRune(got, 0xFFFD) {
			t.Errorf("truncateMessage(maxBytes=%d) contained U+FFFD: %q", maxBytes, got)
		}
		if len(got) > maxBytes {
			t.Errorf("truncateMessage(maxBytes=%d) returned %d bytes, exceeds limit",
				maxBytes, len(got))
		}
	}
}
