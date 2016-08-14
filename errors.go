package gcfg

import (
	"gopkg.in/warnings.v0"
)

func FatalOnly(err error) error {
	return warnings.FatalOnly(err)
}

func isFatal(err error) bool {
	_, ok := err.(extraData)
	return !ok
}

type extraData struct {
	section    string
	subsection *string
	variable   *string
}

func (e extraData) Error() string {
	s := "can't store data at section \"" + e.section + "\""
	if e.subsection != nil {
		s += ", subsection \"" + *e.subsection + "\""
	}
	if e.variable != nil {
		s += ", variable \"" + *e.variable + "\""
	}
	return s
}

var _ error = extraData{}
