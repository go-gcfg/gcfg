package gcfg

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
