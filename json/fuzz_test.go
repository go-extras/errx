package json_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
)

// FuzzMarshal stresses the depth-limited recursion, circular detection, and
// unhashable error handling in errxjson.Marshal. The fuzzer drives a synthetic
// error chain whose shape (depth and contents) is derived from inputs.
func FuzzMarshal(f *testing.F) {
	f.Add("ctx", "msg", "k", "v", uint8(2))
	f.Add("", "", "", "", uint8(0))
	f.Add("outer", "inner cause", "user", "42", uint8(8))

	f.Fuzz(func(t *testing.T, ctx, msg, key, val string, depth uint8) {
		if depth > 24 {
			depth = 24
		}

		// Build a chain that mixes carriers, plain wraps, and unhashable errors
		// to drive different branches of toSerializedError.
		var chain error
		chain = errors.New(msg)
		if depth > 4 {
			// Sprinkle in an unhashable error to exercise the value-pointer path.
			chain = fmt.Errorf("%s: %w", ctx, chain)
			chain = errx.Wrap(ctx, chain, errx.Attrs(key, val))
		}
		for i := uint8(0); i < depth; i++ {
			switch i % 3 {
			case 0:
				chain = errx.Wrap(ctx, chain, errx.NewSentinel("s"))
			case 1:
				chain = fmt.Errorf("%s: %w", ctx, chain)
			default:
				chain = errx.Classify(chain, errx.Attrs(key, val))
			}
		}

		data, marshalErr := errxjson.Marshal(chain)
		if marshalErr != nil {
			t.Fatalf("Marshal failed: %v", marshalErr)
		}
		// Result must always be valid JSON.
		var out errxjson.SerializedError
		if unmarshalErr := json.Unmarshal(data, &out); unmarshalErr != nil {
			t.Fatalf("Unmarshal failed: %v\nbytes=%s", unmarshalErr, string(data))
		}
	})
}
