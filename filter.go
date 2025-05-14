package gomongomon

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type Filter interface {
	Match(document any) bool
}

func NewFilter(filter any) (Filter, error) {
	f, ok := filter.(map[string]any)

	if !ok {
		return nil, fmt.Errorf("Expecting a filter to be a map, got %s", reflect.TypeOf(filter).Kind().String())
	}

	filters := make([]Filter, len(f))
	index := 0
	for key, value := range f {
		switch key {
		case "$and":
			inner, err := newAndFilter(value, key)
			if err != nil {
				return nil, err
			}

			filters[index] = inner
		case "$or":
			inner, err := newOrFilter(value)
			if err != nil {
				return nil, err
			}

			filters[index] = inner
		case "$eq":
			filters[index] = anyFilter{Equal: true}
		case "$ne":
			filters[index] = anyFilter{Equal: false}
		case "$gt":
			fallthrough
		case "$gte":
			fallthrough
		case "$lt":
			fallthrough
		case "$lte":
			if n, ok := value.(int); ok {
				filters[index] = numberToNumberFilter[int]{Target: n, Operation: key}
				continue
			}
			if n, ok := value.(float32); ok {
				filters[index] = numberToNumberFilter[float32]{Target: n, Operation: key}
				continue
			}
			if n, ok := value.(float64); ok {
				filters[index] = numberToNumberFilter[float64]{Target: n, Operation: key}
				continue
			}

			err := fmt.Errorf("Expecting numeric operator target to be `int`, `float32`, `float64`, but got %s",
				reflect.TypeOf(value).Kind().String())

			return nil, err
		}

		index++
	}

	return andFilter{Filters: filters}, nil
}

type wrappedFilter struct {
	Path []string
	Filter Filter
}

func (f wrappedFilter) Match(document any) bool {
	value := getInPath(document, f.Path)
	return f.Filter.Match(value)
}

func newWrappedFilter(filter any, path string) wrappedFilter {
	parts := strings.Split(path, ".")
}

type andFilter struct {
	Filters []Filter
}

func (f andFilter) Match(document any) bool {
	if len(f.Filters) == 0 {
		return false
	}

	for _, sub := range f.Filters {
		if !sub.Match(document) {
			return false
		}
	}

	return true
}

func newAndFilter(filter any, path ...string) (andFilter, error) {
	filters, err := newAndOrFilter(filter)

	if err != nil {
		return andFilter{}, err
	}

	return andFilter{Filters: filters}, nil
}

type orFilter struct {
	Filters []Filter
}

func (f orFilter) Match(document any) bool {
	for _, sub := range f.Filters {
		if sub.Match(document) {
			return true
		}
	}

	return false
}

func newOrFilter(filter any) (orFilter, error) {
	filters, err := newAndOrFilter(filter)

	if err != nil {
		return orFilter{}, err
	}

	return orFilter{Filters: filters}, nil
}

func newAndOrFilter(filter any) ([]Filter, error) {
	m, ok := filter.([]any)

	if !ok {
		err := fmt.Errorf("Filter $and should receive a list, received %s", reflect.TypeOf(filter).Kind().String())
		return nil, err
	}

	f := make([]Filter, len(m))

	for i, inner := range m {
		filter, err := NewFilter(inner)
		if err != nil {
			return nil, err
		}
		f[i] = filter
	}

	return f, nil
}

type anyFilter struct {
	Path []string
	Equal  bool
	Target any
}

func (f anyFilter) Match(document any) bool {
	switch f.Equal {
	case true:
		return document == f.Target
	case false:
		return document != f.Target
	}

	return false
}

type numeric interface {
	int | float32 | float64
}

type numberToNumberFilter[T numeric] struct {
	Operation string
	Target    T
}

func (f numberToNumberFilter[T]) Match(document any) bool {
	n, ok := document.(T)
	if ok {
		return matchNumeric(f, n)
	}

	a, ok := document.(int)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, float64(a))
	}

	return false
}

func matchNumeric[A numeric](f numberToNumberFilter[A], n A) bool {
	switch f.Operation {
	case "$gt":
		return n > f.Target
	case "$gte":
		return n >= f.Target
	case "$lt":
		return n < f.Target
	case "$lte":
		return n <= f.Target
	default:
		log.Printf("Unknown operator %s", f.Operation)
		return false
	}
}

func getInPath(document any, path []string) any {
	for _, key := range path {
		m, ok := document.(map[string]any)

		if !ok {
			return nil
		}

		document = m[key]
	}

	return document
}
