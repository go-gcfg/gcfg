package gcfg

import (
	"testing"
)

var idxerIdxCaseTests = [][]string{
	[]string{"a", "A"},
	[]string{"Hello", "HELLO", "hello"},
}

func TestIdxerIdxCase(t *testing.T) {
	var idxer Idxer
	for _, ns := range idxerIdxCaseTests {
		idxer.add(ns[0])
		for _, ns := range idxerIdxCaseTests {
			idx := idxer.Idx(ns[0])
			for _, n := range ns[1:] {
				if idx2 := idxer.Idx(n); idx != idx2 {
					t.Errorf("Idxer.Idx(%q)==%q; "+
						"want same as Idxer.Idx(%q)==%q", n, idx, ns[0], idx2)
				}
			}
		}
	}
}

var idxerNamesTests = [][]string{
	[]string{},
	[]string{"a"},
	[]string{"a", "b"},
}

func TestIdxerNames(t *testing.T) {
	for _, ns := range idxerNamesTests {
		var idxer Idxer
		vals := make(map[Idx]int)
		for i, n := range ns {
			idxer.add(n)
			vals[idxer.Idx(n)] = i
		}
		ins := idxer.Names()
		if len(ns) != len(ins) {
			t.Errorf("len(Idxer.Names())=%d; want len(ns)=%d; ns=%v, ins=%v",
				len(ins), len(ns), ns, ins)
		}
		seen := make(map[int]struct{})
		for _, n := range ins {
			i := idxer.Idx(n)
			v := vals[i]
			if _, exists := seen[v]; exists {
				t.Errorf("Idxer.Names()=%v contains duplicate; seen=%v",
					ins, seen)
			} else {
				seen[v] = struct{}{}
			}
		}
		if len(seen) != len(ns) {
			t.Errorf("len(seen)=%d; want len(Idxer.Names())=%d; ns=%v",
				len(seen), len(ns), ns)
		}
	}
}
