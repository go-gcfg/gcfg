package gcfg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	reCmnt    = regexp.MustCompile(`^([^;#"]*)[;#].*$`)
	reCmntQ   = regexp.MustCompile(`^([^;#"]*"[^"]*"[^;#"]*)[;#].*$`)
	reBlank   = regexp.MustCompile(`^\s*$`)
	reSect    = regexp.MustCompile(`^\s*\[\s*([^"\s]*)\s*\]\s*$`)
	reSectSub = regexp.MustCompile(`^\s*\[\s*([^"\s]*)\s*"([^"]+)"\s*\]\s*$`)
	reVar     = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*([^"\s]*)\s*$`)
	reVarQ    = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*"([^"\n\\]*)"\s*$`)
	reVarDflt = regexp.MustCompile(`^\s*\b(.*)\b\s*$`)
)

// ReadInto reads gcfg formatted data from reader and sets the values into the
// corresponding fields in config.
//
// Config must be a pointer to a struct.
// Each section corresponds to a struct field in config, and each variable in a
// section corresponds to a data field in the section struct.
// The name of the field must match the name of the section or variable,
// ignoring case.
// Hyphens in section and variable names correspond to underscores in field
// names.
//
// For sections with subsections, the corresponding field in config must be a
// map, rather than a struct, with string keys and pointer-to-struct values.
// Values for subsection variables are stored in the map with the subsection
// name used as the map key.
// (Note that unlike section and variable names, subsection names are case
// sensitive.)
// When using a map, and there is a section with the same section name but
// without a subsection name, its values are stored with the empty string used
// as the key.
//
// The section structs in the config struct may contain arbitrary types.
// For string fields, the (unquoted and unescaped) value string is assigned to
// the field.
// For bool fields, the field is set to true if the value is "true", "yes", "on"
// or "1", and set to false if the value is "false", "no", "off" or "0",
// ignoring case.
// For all other types, fmt.Sscanf with the verb "%v" is used to parse the value
// string and set it to the field.
// This means that built-in Go types are parseable using the standard format,
// and any user-defined type is parseable if it implements the fmt.Scanner
// interface.
// Note that the value is considered invalid unless fmt.Scanner fully consumes
// the value string without error.
//
// ReadInto panics if config is not a pointer to a struct, or if it encounters a
// field that is not of a suitable type (either a struct or a map with string
// keys and pointer-to-struct values).
//
// See ReadStringInto for examples.
//
func ReadInto(config interface{}, reader io.Reader) error {
	r := bufio.NewReader(reader)
	sect, sectsub := "", ""
	lp := []byte{}
	for line := 1; true; line++ {
		l, pre, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return err
		}
		if pre {
			lp = append(lp, l...)
			line--
			continue
		}
		if len(l) > 0 {
			l = append(lp, l...)
			lp = []byte{}
		}
		// exclude comments
		if c := reCmnt.FindSubmatch(l); c != nil {
			l = c[1]
		} else if c := reCmntQ.FindSubmatch(l); c != nil {
			l = c[1]
		}
		if !reBlank.Match(l) {
			// "switch" based on line contents
			if s, ss := reSect.FindSubmatch(l), reSectSub.FindSubmatch(l); //
			s != nil || ss != nil {
				// section
				if s != nil {
					sect, sectsub = string(s[1]), ""
				} else { // ss != nil
					sect, sectsub = string(ss[1]), string(ss[2])
				}
				if sect == "" {
					return fmt.Errorf("empty section name not allowed")
				}
				if ss != nil && sectsub == "" {
					return fmt.Errorf("subsection name \"\" not allowed; " +
						"use [section-name] for blank subsection name")
				}
			} else if v, vq, vd := reVar.FindSubmatch(l),
				reVarQ.FindSubmatch(l), reVarDflt.FindSubmatch(l); //
			v != nil || vq != nil || vd != nil {
				// variable
				if sect == "" {
					return fmt.Errorf("variable must be defined in a section")
				}
				var name, value string
				if v != nil {
					name, value = string(v[1]), string(v[2])
				} else if vq != nil {
					name, value = string(vq[1]), string(vq[2])
				} else { // vd != nil
					name, value = string(vd[1]), defaultValue
				}
				err := set(config, sect, sectsub, name, value)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("invalid line %q", string(l))
			}
		}
		if err == io.EOF {
			break
		}
	}
	return nil
}

// ReadStringInto reads gcfg formatted data from str and sets the values into
// the corresponding fields in config.
// ReadStringInfo is a wrapper for ReadInfo; see ReadInto(config, reader) for
// detailed description of how data is read and set into config.
func ReadStringInto(config interface{}, str string) error {
	r := strings.NewReader(str)
	return ReadInto(config, r)
}

// ReadFileInto reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in config.
// ReadFileInto is a wrapper for ReadInfo; see ReadInto(config, reader) for
// detailed description of how data is read and set into config.
func ReadFileInto(config interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return ReadInto(config, f)
}
