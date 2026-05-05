package require

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	junge "github.com/skipper-ad/junge-checkers"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

func Equal(c *junge.C, expected, actual any, public string, opts ...o.Option) {
	if err := validateComparable(expected, actual); err != nil {
		c.CheckFailedf("Checker failed", "%s: invalid Equal assertion: %v", caller(1), err)
	}
	if !reflect.DeepEqual(expected, actual) {
		finish(c, public, fmt.Sprintf("%s: expected %#v, got %#v", caller(1), expected, actual), opts...)
	}
}

func NotEqual(c *junge.C, expected, actual any, public string, opts ...o.Option) {
	if err := validateComparable(expected, actual); err != nil {
		c.CheckFailedf("Checker failed", "%s: invalid NotEqual assertion: %v", caller(1), err)
	}
	if reflect.DeepEqual(expected, actual) {
		finish(c, public, fmt.Sprintf("%s: values must differ: %#v", caller(1), actual), opts...)
	}
}

func Less[T ordered](c *junge.C, a, b T, public string, opts ...o.Option) {
	if !(a < b) {
		finish(c, public, fmt.Sprintf("%s: expected %v < %v", caller(1), a, b), opts...)
	}
}

func LessOrEqual[T ordered](c *junge.C, a, b T, public string, opts ...o.Option) {
	if !(a <= b) {
		finish(c, public, fmt.Sprintf("%s: expected %v <= %v", caller(1), a, b), opts...)
	}
}

func Greater[T ordered](c *junge.C, a, b T, public string, opts ...o.Option) {
	if !(a > b) {
		finish(c, public, fmt.Sprintf("%s: expected %v > %v", caller(1), a, b), opts...)
	}
}

func GreaterOrEqual[T ordered](c *junge.C, a, b T, public string, opts ...o.Option) {
	if !(a >= b) {
		finish(c, public, fmt.Sprintf("%s: expected %v >= %v", caller(1), a, b), opts...)
	}
}

func True(c *junge.C, value bool, public string, opts ...o.Option) {
	if !value {
		finish(c, public, fmt.Sprintf("%s: expected true", caller(1)), opts...)
	}
}

func False(c *junge.C, value bool, public string, opts ...o.Option) {
	if value {
		finish(c, public, fmt.Sprintf("%s: expected false", caller(1)), opts...)
	}
}

func NoError(c *junge.C, err error, public string, opts ...o.Option) {
	if err != nil {
		finish(c, public, fmt.Sprintf("%s: unexpected error: %v", caller(1), err), opts...)
	}
}

func Error(c *junge.C, err error, public string, opts ...o.Option) {
	if err == nil {
		finish(c, public, fmt.Sprintf("%s: expected error, got nil", caller(1)), opts...)
	}
}

func Nil(c *junge.C, value any, public string, opts ...o.Option) {
	if !isNil(value) {
		finish(c, public, fmt.Sprintf("%s: expected nil, got %#v", caller(1), value), opts...)
	}
}

func NotNil(c *junge.C, value any, public string, opts ...o.Option) {
	if isNil(value) {
		finish(c, public, fmt.Sprintf("%s: expected not nil", caller(1)), opts...)
	}
}

func Contains(c *junge.C, haystack, needle string, public string, opts ...o.Option) {
	if !strings.Contains(haystack, needle) {
		finish(c, public, fmt.Sprintf("%s: expected %q to contain %q", caller(1), haystack, needle), opts...)
	}
}

func NotContains(c *junge.C, haystack, needle string, public string, opts ...o.Option) {
	if strings.Contains(haystack, needle) {
		finish(c, public, fmt.Sprintf("%s: expected %q not to contain %q", caller(1), haystack, needle), opts...)
	}
}

func In[T comparable](c *junge.C, needle T, haystack []T, public string, opts ...o.Option) {
	for _, value := range haystack {
		if value == needle {
			return
		}
	}
	finish(c, public, fmt.Sprintf("%s: expected %#v to be in %#v", caller(1), needle, haystack), opts...)
}

func NotIn[T comparable](c *junge.C, needle T, haystack []T, public string, opts ...o.Option) {
	for _, value := range haystack {
		if value == needle {
			finish(c, public, fmt.Sprintf("%s: expected %#v not to be in %#v", caller(1), needle, haystack), opts...)
		}
	}
}

func finish(c *junge.C, public, private string, opts ...o.Option) {
	info := o.GetExitInfo(public, private, opts...)
	c.Finish(info.Status, info.Public, info.Private)
}

func caller(depth int) string {
	_, file, line, ok := runtime.Caller(depth + 1)
	if !ok {
		return "<unknown>:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

func validateComparable(expected, actual any) error {
	if expected == nil || actual == nil {
		return nil
	}
	if reflect.TypeOf(expected).Kind() == reflect.Func || reflect.TypeOf(actual).Kind() == reflect.Func {
		return errors.New("cannot compare function values")
	}
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
