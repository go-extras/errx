package stacktrace_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// TestWrap_DoesNotMutateCallerSlice is a regression test for the defensive copy in
// stacktrace.Wrap. When the caller spreads a slice with spare capacity, the
// previous implementation could write the captured trace into the caller's
// backing array; the fix copies into a fresh slice instead.
func TestWrap_DoesNotMutateCallerSlice(t *testing.T) {
	c1 := errx.NewSentinel("c1")
	c2 := errx.NewSentinel("c2")

	cls := make([]errx.Classified, 0, 4) // spare capacity is the trigger
	cls = append(cls, c1, c2)

	// Snapshot what cls and its backing array look like.
	snapshot := append([]errx.Classified(nil), cls...)
	origLen := len(cls)
	origCap := cap(cls)

	base := errors.New("base")
	_ = stacktrace.Wrap("op", base, cls...)

	if len(cls) != origLen {
		t.Errorf("len(cls) changed: was %d, now %d", origLen, len(cls))
	}
	if cap(cls) != origCap {
		t.Errorf("cap(cls) changed: was %d, now %d", origCap, cap(cls))
	}
	for i := range snapshot {
		if cls[i] != snapshot[i] {
			t.Errorf("cls[%d] changed: was %#v, now %#v", i, snapshot[i], cls[i])
		}
	}

	// Also poke past len(cls) within cap: previous implementation would have
	// written the captured trace into cls[len(cls):cap(cls)].
	tail := cls[:cap(cls)]
	for i := origLen; i < origCap; i++ {
		if tail[i] != nil {
			t.Errorf("cls backing array beyond len was mutated at index %d: %#v", i, tail[i])
		}
	}
}

// TestClassify_DoesNotMutateCallerSlice mirrors the regression test for stacktrace.Classify.
func TestClassify_DoesNotMutateCallerSlice(t *testing.T) {
	c1 := errx.NewSentinel("c1")
	cls := make([]errx.Classified, 0, 3)
	cls = append(cls, c1)

	origLen := len(cls)
	origCap := cap(cls)

	base := errors.New("base")
	_ = stacktrace.Classify(base, cls...)

	if len(cls) != origLen {
		t.Errorf("len(cls) changed: was %d, now %d", origLen, len(cls))
	}
	if cap(cls) != origCap {
		t.Errorf("cap(cls) changed: was %d, now %d", origCap, cap(cls))
	}
	tail := cls[:cap(cls)]
	for i := origLen; i < origCap; i++ {
		if tail[i] != nil {
			t.Errorf("cls backing array beyond len was mutated at index %d: %#v", i, tail[i])
		}
	}
}

// TestClassifyNew_DoesNotMutateCallerSlice mirrors the regression test for stacktrace.ClassifyNew.
func TestClassifyNew_DoesNotMutateCallerSlice(t *testing.T) {
	c1 := errx.NewSentinel("c1")
	cls := make([]errx.Classified, 0, 2)
	cls = append(cls, c1)

	origLen := len(cls)
	origCap := cap(cls)

	_ = stacktrace.ClassifyNew("oops", cls...)

	if len(cls) != origLen {
		t.Errorf("len(cls) changed: was %d, now %d", origLen, len(cls))
	}
	if cap(cls) != origCap {
		t.Errorf("cap(cls) changed: was %d, now %d", origCap, cap(cls))
	}
	tail := cls[:cap(cls)]
	for i := origLen; i < origCap; i++ {
		if tail[i] != nil {
			t.Errorf("cls backing array beyond len was mutated at index %d: %#v", i, tail[i])
		}
	}
}
