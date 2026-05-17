package errx_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/go-extras/errx"
)

// --- Item 2: concurrency / race coverage ---------------------------------------------------------

// TestConcurrent_ReadOnly exercises a shared errx error from many goroutines to
// validate the "read-only after construction" contract under -race.
func TestConcurrent_ReadOnly(t *testing.T) {
	t.Parallel()

	err := errx.Wrap("ctx", errors.New("base"),
		errx.NewDisplayable("user msg"),
		errx.Attrs("k", "v"),
		errx.NewSentinel("sentinel"))

	const (
		goroutines = 32
		iterations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = errx.ExtractAttrs(err)
				_ = errx.DisplayText(err)
				_ = errx.IsDisplayable(err)
				_ = errx.HasAttrs(err)
				_ = err.Error()
			}
		}()
	}
	wg.Wait()
}

// --- Item 3: deep / wide / diamond chain coverage --------------------------------------------------

// TestDeepWrapChain verifies that a very deep Wrap chain remains traversable
// (Error/Is/ExtractAttrs all terminate without stack issues).
func TestDeepWrapChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deep-chain stress test in -short mode")
	}

	const depth = 200
	sentinel := errx.NewSentinel("deep-sentinel")

	err := errx.Classify(errors.New("base"), sentinel)
	for i := 0; i < depth; i++ {
		err = errx.Wrap(fmt.Sprintf("level %d", i), err)
	}

	if !errors.Is(err, sentinel) {
		t.Error("deep chain should still match sentinel")
	}
	// Should not panic or hang.
	_ = err.Error()
}

// TestWideClassifications verifies a carrier with many classifications
// behaves correctly (errors.Is matches all of them).
func TestWideClassifications(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wide-classifications stress test in -short mode")
	}

	const count = 150
	sentinels := make([]errx.Classified, count)
	for i := 0; i < count; i++ {
		sentinels[i] = errx.NewSentinel(fmt.Sprintf("s%d", i))
	}

	err := errx.Classify(errors.New("base"), sentinels...)

	for i, s := range sentinels {
		if !errors.Is(err, s) {
			t.Errorf("expected match for sentinel %d", i)
		}
	}
}

// TestWideAttributes verifies that an attributed with 1000+ attrs is
// extracted in stable order.
func TestWideAttributes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wide-attributes stress test in -short mode")
	}

	const count = 1200
	raw := make([]any, 0, count*2)
	for i := 0; i < count; i++ {
		raw = append(raw, fmt.Sprintf("k%d", i), i)
	}

	err := errx.Attrs(raw...)
	attrs := errx.ExtractAttrs(err)

	if len(attrs) != count {
		t.Fatalf("expected %d attrs, got %d", count, len(attrs))
	}
	// Stable order check on a few sample positions.
	for _, i := range []int{0, count / 2, count - 1} {
		if attrs[i].Key != fmt.Sprintf("k%d", i) || attrs[i].Value != i {
			t.Errorf("attrs[%d]=%+v; expected k%d=%d", i, attrs[i], i, i)
		}
	}
}

// TestDiamondAttributedDedup verifies the same *attributed instance reached via
// two different carrier paths is only counted once by ExtractAttrs.
func TestDiamondAttributedDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping diamond-attribute dedup stress test in -short mode")
	}

	shared := errx.Attrs("shared_key", "shared_value")

	// Build two carriers that both reference the same *attributed and combine
	// them in a multi-error to form a diamond.
	branch1 := errx.Classify(errors.New("b1"), shared, errx.NewSentinel("s1"))
	branch2 := errx.Classify(errors.New("b2"), shared, errx.NewSentinel("s2"))

	diamond := &diamondMultiError{errs: []error{branch1, branch2}}

	attrs := errx.ExtractAttrs(diamond)

	// The shared attributed instance must contribute its attrs exactly once,
	// even though it is reachable via two distinct carrier paths.
	count := 0
	for _, a := range attrs {
		if a.Key == "shared_key" && a.Value == "shared_value" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected shared attributed contribution exactly once, got %d (all attrs: %v)", count, attrs)
	}
}

// diamondMultiError is a local multi-error helper used by the diamond test.
type diamondMultiError struct {
	errs []error
}

func (*diamondMultiError) Error() string     { return "diamond" }
func (m *diamondMultiError) Unwrap() []error { return m.errs }

// --- Item 4: sentinel.As parent traversal ----------------------------------------------------------

// customSentinelError is a custom error type used to test that sentinel.As
// walks its parents list (not just the carrier).
type customSentinelError struct {
	tag string
}

func (e *customSentinelError) Error() string    { return "custom: " + e.tag }
func (*customSentinelError) IsClassified() bool { return true }

// errAsParent wraps a custom error so that errors.As can find it via the
// sentinel.parents slice (sentinel itself is the only path).
type errAsParent struct {
	custom *customSentinelError
}

func (e *errAsParent) Error() string    { return e.custom.Error() }
func (e *errAsParent) Unwrap() error    { return e.custom }
func (*errAsParent) IsClassified() bool { return true }

// TestSentinel_AsParentTraversal verifies that errors.As traverses sentinel
// parents when looking for a typed match. Without the parents-loop in
// (*sentinel).As, this test fails to extract *customSentinelError.
func TestSentinel_AsParentTraversal(t *testing.T) {
	customErr := &customSentinelError{tag: "auth"}
	parent := &errAsParent{custom: customErr}

	// Build a sentinel whose only parent is our custom Classified.
	child := errx.NewSentinel("child", parent)

	// errors.As should walk the sentinel's parents and locate the custom error.
	var target *customSentinelError
	if !errors.As(child, &target) {
		t.Fatal("expected errors.As to find *customSentinelError via sentinel.parents")
	}
	if target.tag != "auth" {
		t.Errorf("expected tag 'auth', got %q", target.tag)
	}
}

// --- Item 5: deprecated WithAttrs alias ------------------------------------------------------------

// TestWithAttrs_DeprecatedAlias explicitly invokes the deprecated WithAttrs
// alias (NOT Attrs) to guard against accidental breakage (e.g. self-recursion
// or nil return).
func TestWithAttrs_DeprecatedAlias(t *testing.T) {
	err := errx.WithAttrs(
		errx.Attr{Key: "alias_key", Value: "alias_value"},
		"k2", 42,
	)
	if err == nil {
		t.Fatal("WithAttrs returned nil")
	}
	if !errx.HasAttrs(err) {
		t.Fatal("WithAttrs result should report HasAttrs == true")
	}

	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
	if attrs[0].Key != "alias_key" || attrs[0].Value != "alias_value" {
		t.Errorf("attrs[0] = %+v", attrs[0])
	}
	if attrs[1].Key != "k2" || attrs[1].Value != 42 {
		t.Errorf("attrs[1] = %+v", attrs[1])
	}
}

// --- Item 6: string-producing methods coverage -----------------------------------------------------

// TestAttr_String asserts the exact formatting of Attr.String().
func TestAttr_String(t *testing.T) {
	a := errx.Attr{Key: "user_id", Value: 123}
	got := a.String()
	want := "user_id=123"
	if got != want {
		t.Errorf("Attr.String() = %q, want %q", got, want)
	}
}

// TestAttrList_String asserts the exact formatting of AttrList.String().
func TestAttrList_String(t *testing.T) {
	tests := []struct {
		name string
		al   errx.AttrList
		want string
	}{
		{
			name: "empty",
			al:   errx.AttrList{},
			want: "",
		},
		{
			name: "single",
			al:   errx.AttrList{{Key: "k", Value: "v"}},
			want: "k=v",
		},
		{
			name: "multiple",
			al: errx.AttrList{
				{Key: "user_id", Value: 1},
				{Key: "action", Value: "delete"},
			},
			want: "user_id=1 action=delete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.al.String(); got != tt.want {
				t.Errorf("AttrList.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAttributed_ErrorEmpty asserts the exact literal returned by
// attributed.Error() when the attribute list is empty. Pinning this string
// prevents silent removal of the "(empty attribute list)" sentinel.
func TestAttributed_ErrorEmpty(t *testing.T) {
	err := errx.Attrs()
	if err.Error() != "(empty attribute list)" {
		t.Errorf("attributed.Error() with empty list = %q, want %q",
			err.Error(), "(empty attribute list)")
	}
}

// TestAttributed_ErrorNonEmpty asserts attributed.Error() returns the
// joined "k=v" form for non-empty attribute lists.
func TestAttributed_ErrorNonEmpty(t *testing.T) {
	err := errx.Attrs("user_id", 1, "action", "delete")
	want := "user_id=1 action=delete"
	if err.Error() != want {
		t.Errorf("attributed.Error() = %q, want %q", err.Error(), want)
	}
}
