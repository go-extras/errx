package json

import (
	"errors"
	"testing"

	"github.com/go-extras/errx/internal/errptr"
)

// TestVisitedIdentity checks cycle detection and path-local cleanup separately
// from Marshal's chain-wide metadata extractors, which do not all support cycles.
func TestVisitedIdentity(t *testing.T) {
	var visited visitedSet
	zero := errptr.Get(nil)
	for range 2 {
		if enterVisited(&visited, zero) || visited != nil {
			t.Fatal("zero identity must not be tracked")
		}
	}
	exitVisited(&visited, zero)
	err := errors.New("cycle")
	id := errptr.Get(err)
	if enterVisited(&visited, id) {
		t.Fatal("first visit must not be a cycle")
	}
	if !enterVisited(&visited, errptr.Get(err)) {
		t.Fatal("revisiting the same error on a path must be a cycle")
	}
	exitVisited(&visited, id)
	if enterVisited(&visited, id) {
		t.Fatal("the same error on a later path must not be a cycle")
	}
}
