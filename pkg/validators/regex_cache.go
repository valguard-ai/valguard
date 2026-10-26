package validators

import (
	"regexp"
	"sync"
)

var regexCache sync.Map // pattern string -> *regexp.Regexp

// CompileCached returns a cached compiled regexp for pattern.
func CompileCached(pattern string) (*regexp.Regexp, error) {
	return compileCached(pattern)
}

func compileCached(pattern string) (*regexp.Regexp, error) {
	if v, ok := regexCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	actual, _ := regexCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp), nil
}

// MustCompileCached panics when pattern is invalid.
func MustCompileCached(pattern string) *regexp.Regexp {
	return mustCompileCached(pattern)
}

func mustCompileCached(pattern string) *regexp.Regexp {
	re, err := compileCached(pattern)
	if err != nil {
		panic("validators: invalid cached pattern: " + pattern)
	}
	return re
}
