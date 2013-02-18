package gcfg

import (
	"strings"
	"testing"
)

type Config1 struct {
	Section struct {
		Int int
	}
}

var testsIssue1 = []string{
	"[section]\nint=X",
	"[section]\nint=",
	"[section]\nint=1A",
}

// value parse error message shouldn't include reflect internals
func TestIssue1(t *testing.T) {
	for i, tt := range testsIssue1 {
		var c Config1
		err := ReadStringInto(&c, tt)
		switch {
		case err == nil:
			t.Errorf("%d: ok; wanted error", i)
		case strings.Contains(err.Error(), "reflect"):
			t.Errorf("%d: error message includes reflect internals: %v", i, err)
		default:
			t.Logf("%d: %v", i, err)
		}
	}
}
