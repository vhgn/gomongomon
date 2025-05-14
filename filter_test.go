package gomongomon_test

import (
	"errors"
	gm "gomongomon"
	"testing"
)

type A []any
type M map[string]any

func TestFilter(t *testing.T) {
	t.Run("Numeric operations", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"age": M{
				"$gte": 18,
			},
			"height": M{
				"$lt": 180,
			},
		})

		errors.Unwrap(e)

		f.Match(M{
			"age": 17.5,
		})
	})
}
