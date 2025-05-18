package gomongomon_test

import (
	gm "gomongomon"
	"testing"
)

type A = []any
type M = map[string]any

func TestFilter(t *testing.T) {
	t.Run("Numeric operations", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"age": M{
				"$gte": 16,
			},
			"height": M{
				"$lt": 180,
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m := f.Match(M{
			"age":    17.5,
			"height": 170,
		})

		if !m {
			t.Error("Age 17.5 did not match")
		}

		m = f.Match(M{})

		if m {
			t.Error("Empty matches")
		}

		m = f.Match(M{
			"age":    18,
			"height": 180,
		})

		if m {
			t.Error("$lte should not match")
		}
	})

	t.Run("Boolean operations", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"$and": A{
				M{
					"$or": A{
						M{
							"a.b": M{
								"$eq": 1,
							},
						},
						M{
							"a.b": 2,
						},
					},
				},
				M{
					"a.c": 3,
				},
			},
		})

		if e != nil {
			t.Error(e)
		}

		m := f.Match(M{
			"a": M{
				"b": 2,
				"c": 3,
			},
		})

		if !m {
			t.Error("Coult not match deep object")
		}
	})
}

func Assert(t *testing.T, e error) {
	if e != nil {
		t.Log(e)
		t.FailNow()
	}
}
