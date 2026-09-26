package json_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

// traceClassification represents an external trace carrier, including one
// whose only text would otherwise be mistaken for a sentinel.
type traceClassification struct{ frames []stacktrace.Frame }

func (traceClassification) Error() string                { return "external trace" }
func (traceClassification) IsClassified() bool           { return true }
func (c traceClassification) Frames() []stacktrace.Frame { return c.frames }

// emptyAttributeClassification exercises kind detection for external carriers.
type emptyAttributeClassification struct{}

func (emptyAttributeClassification) Error() string      { return "external attributes" }
func (emptyAttributeClassification) IsClassified() bool { return true }
func (emptyAttributeClassification) Attrs() []errx.Attr { return nil }

// TestMarshal_ClassificationKinds checks exact sentinel output for metadata
// carriers and for sentinels whose parents carry metadata.
func TestMarshal_ClassificationKinds(t *testing.T) {
	attrs := errx.Attrs("key", "value")
	trace := stacktrace.Here()
	frames := []stacktrace.Frame{{File: "db.go", Line: 42, Function: "db.Query"}}
	wantAttrs := []errxjson.SerializedAttr{{Key: "key", Value: "value"}}
	tests := []struct {
		name       string
		cls        errx.Classified
		sentinels  []string
		attributes []errxjson.SerializedAttr
		trace      bool
		display    string
	}{
		{name: "empty attrs", cls: errx.Attrs()},
		{name: "nil attr map", cls: errx.FromAttrMap(nil)},
		{name: "empty attr map", cls: errx.FromAttrMap(make(errx.AttrMap))},
		{name: "nonempty attrs", cls: attrs, attributes: wantAttrs},
		{name: "internal trace", cls: trace, trace: true},
		{name: "external nil trace", cls: traceClassification{}},
		{name: "external empty trace", cls: traceClassification{frames: make([]stacktrace.Frame, 0)}},
		{name: "external nonempty trace", cls: traceClassification{frames: frames}, trace: true},
		{name: "external empty attrs", cls: emptyAttributeClassification{}},
		{name: "sentinel", cls: errx.NewSentinel("child"), sentinels: []string{"child"}},
		{name: "attributed parent", cls: errx.NewSentinel("child", attrs), sentinels: []string{"child"}, attributes: wantAttrs},
		{name: "traced parent", cls: errx.NewSentinel("child", trace), sentinels: []string{"child"}, trace: true},
		{name: "displayable", cls: errx.NewDisplayable("safe"), display: "safe"},
		{name: "displayable parent", cls: errx.NewSentinel("child", errx.NewDisplayable("safe")), display: "safe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := errors.New("base")
			mixedSentinels := append([]string{"real sentinel"}, tt.sentinels...)
			shapes := []struct {
				name      string
				err       error
				sentinels []string
			}{
				{"classified", errx.Classify(base, tt.cls), tt.sentinels},
				{"wrapped", errx.Wrap("op", base, tt.cls), tt.sentinels},
				{"standalone", tt.cls, tt.sentinels},
				{"mixed", errx.Classify(base, errx.NewSentinel("real sentinel"), tt.cls), mixedSentinels},
			}
			for _, shape := range shapes {
				t.Run(shape.name, func(t *testing.T) {
					data, err := errxjson.Marshal(shape.err)
					if err != nil {
						t.Fatal(err)
					}
					var got errxjson.SerializedError
					if err := json.Unmarshal(data, &got); err != nil {
						t.Fatal(err)
					}
					if got.Message != shape.err.Error() {
						t.Errorf("Message = %q, want %q", got.Message, shape.err.Error())
					}
					if !reflect.DeepEqual(got.Sentinels, shape.sentinels) {
						t.Errorf("Sentinels = %q, want %q; JSON: %s", got.Sentinels, shape.sentinels, data)
					}
					if !reflect.DeepEqual(got.Attributes, tt.attributes) {
						t.Errorf("Attributes = %v, want %v", got.Attributes, tt.attributes)
					}
					if hasTrace := len(got.StackTrace) > 0; hasTrace != tt.trace {
						t.Errorf("trace present = %v, want %v", hasTrace, tt.trace)
					}
					if got.DisplayText != tt.display {
						t.Errorf("DisplayText = %q, want %q", got.DisplayText, tt.display)
					}
					var fields map[string]json.RawMessage
					if err := json.Unmarshal(data, &fields); err != nil {
						t.Fatal(err)
					}
					if _, present := fields["sentinels"]; present != (len(shape.sentinels) > 0) {
						t.Errorf("sentinels field presence is incorrect: %s", data)
					}
				})
			}
		})
	}
}
