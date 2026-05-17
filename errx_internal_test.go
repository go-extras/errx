package errx

import (
	"testing"
)

// TestNonNilClassifications_NoNilsReturnsSameSlice verifies that when the
// input contains no nil entries, nonNilClassifications returns the same
// backing array (no copy, no allocation).
func TestNonNilClassifications_NoNilsReturnsSameSlice(t *testing.T) {
	cs := []Classified{
		NewSentinel("a"),
		NewSentinel("b"),
		NewSentinel("c"),
	}

	out := nonNilClassifications(cs)

	if len(out) != len(cs) {
		t.Fatalf("expected len %d, got %d", len(cs), len(out))
	}
	if &cs[0] != &out[0] {
		t.Fatalf("expected nonNilClassifications to return the same backing array on the happy path")
	}
}

// TestNonNilClassifications_AllocatesOnlyWhenNeeded confirms that the happy
// path (no nil entries) performs zero allocations.
func TestNonNilClassifications_AllocatesOnlyWhenNeeded(t *testing.T) {
	cs := []Classified{
		NewSentinel("a"),
		NewSentinel("b"),
		NewSentinel("c"),
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = nonNilClassifications(cs)
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocs on the happy path, got %v", allocs)
	}
}
