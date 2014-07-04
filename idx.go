package gcfg

import (
	"strings"
)

// Idx represents a handle to use as map key when looking up variable names in a
// case-insensitive manner.
type Idx idx

type idx struct {
	n string
}

// Idxer implements case-insensitive lookup of variables in a section.
type Idxer struct {
	names map[string]struct{}
}

// Idx returns the Idx for the variable n, matched case-insensitively.
// In case of no match, the Idx returned is one that does not exist in the map.
func (i Idxer) Idx(n string) Idx {
	if i.names == nil {
		return Idx{}
	}
	for in := range i.names {
		if strings.EqualFold(n, in) {
			return Idx{in}
		}
	}
	return Idx{}
}

// Names returns the variable names for the section. The case and order of names
// is undefined.
func (i Idxer) Names() []string {
	if i.names == nil {
		return nil
	}
	l := make([]string, 0, len(i.names))
	for n := range i.names {
		l = append(l, n)
	}
	return l
}

// add adds n to Idxer. Checking for duplicates is the caller's responsibility.
func (i *Idxer) add(n string) {
	if i.names == nil {
		i.names = make(map[string]struct{})
	}
	i.names[n] = struct{}{}
}
