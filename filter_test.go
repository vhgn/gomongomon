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

	t.Run("Exists filter", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"name": M{
				"$exists": true,
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m := f.Match(M{
			"name": "John",
		})

		if !m {
			t.Error("Exists does not work")
		}
	})

	t.Run("In and Nin filter", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"name": M{
				"$in": A{"John", "Jane"},
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m := f.Match(M{
			"name": "John",
		})

		if !m {
			t.Error("In does not match")
		}

		m = f.Match(M{
			"name": "Jake",
		})

		if m {
			t.Error("In does not reject")
		}

		f, e = gm.NewFilter(M{
			"name": M{
				"$nin": A{"Ash", "Zoe"},
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m = f.Match(M{
			"name": "John",
		})

		if !m {
			t.Error("Nin does not match")
		}

		m = f.Match(M{
			"name": "Ash",
		})

		if m {
			t.Error("Nin does not reject")
		}
	})

	t.Run("Not filter works", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"a": M{
				"$not": M{
					"$eq": 1,
				},
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m := f.Match(M{
			"a": 1,
		})

		if m {
			t.Error("Not filter does not reject")
		}
	})

	t.Run("Regex filter works", func(t *testing.T) {
		f, e := gm.NewFilter(M{
			"name": M{
				"$regex": "^_.*",
			},
		})

		if e != nil {
			t.Log(e)
			t.FailNow()
		}

		m := f.Match(M{
			"name": "_abc",
		})

		if !m {
			t.Error("Regex does not accept")
		}

		m = f.Match(M{
			"name": "abc",
		})

		if m {
			t.Error("Regex does not reject")
		}
	})
}

func Assert(t *testing.T, e error) {
	if e != nil {
		t.Log(e)
		t.FailNow()
	}
}
