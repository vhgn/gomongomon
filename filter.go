package gomongomon

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"time"
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
			inner, err := newAndFilter(value)
			if err != nil {
				return nil, err
			}

			filters[index] = inner
		case "$nor":
			inner, err := newOrFilter(true, value)
			if err != nil {
				return nil, err
			}

			filters[index] = inner
		case "$or":
			inner, err := newOrFilter(false, value)
			if err != nil {
				return nil, err
			}

			filters[index] = inner
		default:
			f, err := newWrappedFilter(key, value)
			if err != nil {
				return nil, err
			}
			filters[index] = f
		}

		index++
	}

	if len(filters) == 1 {
		return filters[0], nil
	} else {
		return andFilter{Filters: filters}, nil
	}
}

type wrappedFilter struct {
	Path   []string
	Filter Filter
}

func (f wrappedFilter) Match(document any) bool {
	value := getInPath(document, f.Path)
	return f.Filter.Match(value)
}

func newWrappedFilter(path string, filterMap any) (Filter, error) {
	parts := strings.Split(path, ".")
	f, isMap := filterMap.(map[string]any)
	if !isMap {
		target := filterMap
		f := anyFilter{Equal: true, Target: target}
		return wrappedFilter{Path: parts, Filter: f}, nil
	}
	filters := make([]Filter, len(f))
	index := 0
	for key, value := range f {
		var filter Filter
		switch key {
		case "$all":
			f, err := newArrayFilter(true, value)
			if err != nil {
				return nil, err
			}

			filter = f
		case "$elemMatch":
			f, err := newArrayFilter(false, value)
			if err != nil {
				return nil, err
			}

			filter = f
		case "$not":
			f, err := newWrappedFilter("", value)
			if err != nil {
				return nil, err
			}

			filter = notFilter{Filter: f}
		case "$nin":
			f, err := newInFilter(false, value)
			if err != nil {
				return nil, err
			}
			filter = f
		case "$in":
			f, err := newInFilter(true, value)
			if err != nil {
				return nil, err
			}
			filter = f
		case "$regex":
			r, ok := value.(string)
			if !ok {
				err := fmt.Errorf("Expecting string for regex filter, got %s", reflect.TypeOf(value))
				return nil, err
			}
			reg, err := regexp.Compile(r)
			if err != nil {
				return nil, err
			}
			filter = regexFilter{Regex: reg}
		case "$type":
			t, ok := value.(string)
			if !ok {
				err := fmt.Errorf("$type filter value should be string")
				return nil, err
			}
			filter = typeFilter{Type: t}
		case "$eq":
			filter = anyFilter{Equal: true, Target: value}
		case "$ne":
			filter = anyFilter{Equal: false, Target: value}
		case "$gt":
			fallthrough
		case "$gte":
			fallthrough
		case "$lt":
			fallthrough
		case "$lte":
			if n, ok := value.(int); ok {
				filter = numberToNumberFilter[int]{Target: n, Operation: key}
				break
			}
			if n, ok := value.(int64); ok {
				filter = numberToNumberFilter[int64]{Target: n, Operation: key}
				break
			}
			if n, ok := value.(float32); ok {
				filter = numberToNumberFilter[float32]{Target: n, Operation: key}
				break
			}
			if n, ok := value.(float64); ok {
				filter = numberToNumberFilter[float64]{Target: n, Operation: key}
				break
			}
			if n, ok := value.(time.Time); ok {
				filter = numberToNumberFilter[int64]{Target: n.Unix(), Operation: key}
				break
			}

			err := fmt.Errorf("Expecting numeric operator target to be `int`, `float32`, `float64`, but got %s",
				reflect.TypeOf(value).Kind().String())

			return nil, err
		case "$exists":
			if exists, ok := value.(bool); ok {
				filter = existsFilter{Exists: exists}
			} else {
				err := fmt.Errorf("$exists should specify a boolean, specified %v", value)
				return nil, err
			}

		default:
			err := fmt.Errorf("Filter %s not supported", key)
			return nil, err
		}
		filters[index] = filter
		index++
	}

	if len(filters) == 1 {
		return wrappedFilter{Path: parts, Filter: filters[0]}, nil
	} else {
		return wrappedFilter{Path: parts, Filter: andFilter{Filters: filters}}, nil
	}
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

func newAndFilter(filter any) (Filter, error) {
	filters, err := newAndOrFilter(filter)

	if err != nil {
		return andFilter{}, err
	}

	if len(filters) == 1 {
		return filters[0], nil
	} else {
		return andFilter{Filters: filters}, nil
	}
}

type orFilter struct {
	Invert  bool
	Filters []Filter
}

func (f orFilter) Match(document any) bool {
	for _, sub := range f.Filters {
		if sub.Match(document) {
			return !f.Invert
		}
	}

	return f.Invert
}

func newOrFilter(invert bool, filter any) (Filter, error) {
	filters, err := newAndOrFilter(filter)

	if err != nil {
		return orFilter{}, err
	}

	if len(filters) == 1 {
		return filters[0], nil
	} else {
		return orFilter{Invert: invert, Filters: filters}, nil
	}
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
	Equal  bool
	Target any
}

func (f anyFilter) Match(document any) bool {
	m := reflect.DeepEqual(document, f.Target)
	switch f.Equal {
	case true:
		return m
	case false:
		return !m
	}

	return false
}

type typeFilter struct {
	Type string
}

func (f typeFilter) Match(document any) bool {
	switch f.Type {
	case "null":
		return document == nil
	case "string":
		_, ok := document.(string)
		return ok
	default:
		return false
	}
}

type existsFilter struct {
	Exists bool
}

func (f existsFilter) Match(document any) bool {
	if document == nil {
		return !f.Exists
	} else {
		return f.Exists
	}
}

type arrayFilter struct {
	All    bool
	Filter Filter
}

func (f arrayFilter) Match(document any) bool {
	a, ok := document.([]any)
	if !ok {
		return false
	}

	if len(a) == 0 {
		return f.All
	}

	for _, elem := range a {
		m := f.Filter.Match(elem)
		if f.All {
			if !m {
				return false
			}
		} else {
			if m {
				return true
			}
		}
	}

	return f.All
}

func newArrayFilter(all bool, filter any) (Filter, error) {
	f, err := newWrappedFilter("", filter)
	if err != nil {
		return nil, err
	}

	return arrayFilter{All: all, Filter: f}, nil
}

func newInFilter(in bool, values any) (Filter, error) {
	v, ok := values.([]any)
	if !ok {
		err := fmt.Errorf("$in and $nin expect array values, got %s", reflect.TypeOf(values))
		return nil, err
	}

	filters := make([]Filter, len(v))
	for i, f := range v {
		filters[i] = anyFilter{
			Equal:  true,
			Target: f,
		}
	}

	f := orFilter{Filters: filters, Invert: !in}

	return f, nil
}

type notFilter struct {
	Filter Filter
}

func (f notFilter) Match(document any) bool {
	m := f.Filter.Match(document)
	return !m
}

type regexFilter struct {
	Regex *regexp.Regexp
}

func (f regexFilter) Match(document any) bool {
	v, ok := document.(string)
	if !ok {
		return false
	}

	return f.Regex.MatchString(v)
}

type numeric interface {
	int | int64 | float32 | float64
}

type numberToNumberFilter[T numeric] struct {
	Operation string
	Target    T
}

func (f numberToNumberFilter[T]) Match(document any) bool {
	v := reflect.ValueOf(document)
	if v.Kind() == reflect.Ptr {
		if !v.IsValid() {
			return false
		}
		return f.Match(v.Elem().Interface())
	}

	n, ok := document.(T)
	if ok {
		return matchNumeric(f, n)
	}

	a, ok := document.(int)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, float64(a))
	}

	b, ok := document.(float32)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, float64(b))
	}

	c, ok := document.(float64)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, c)
	}

	d, ok := document.(int64)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, float64(d))
	}

	e, ok := document.(time.Time)
	if ok {
		return matchNumeric(numberToNumberFilter[float64]{Operation: f.Operation, Target: float64(f.Target)}, float64(e.Unix()))
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
		if key == "" {
			break
		}

		m, ok := document.(map[string]any)

		if !ok {
			return nil
		}

		document = m[key]
	}

	return document
}
