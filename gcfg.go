// Package gcfg reads "gitconfig-like" text-based configuration files with
// "name=value" pairs grouped into sections (gcfg files). Support for modifying
// and/or exporting such files may be added later.
//
// This package is a work in progress, and both the supported file format and
// the API is subject to change.
//
// The syntax is based on that used by git config:
// http://git-scm.com/docs/git-config#_syntax .
// Note that the gcfg syntax may diverge from that of git config in the future
// to a limited degree. Current differences (apart from TODOs listed below) are:
//  - gcfg files must use UTF-8 encoding (for now)
//  - include is not supported (and not planned) 
//
// The package may be usable for handling some of the various "INI file" formats
// used by some programs and libraries, but achieving or maintaining
// compatibility with any of those is not a primary concern.
//
// TODO: besides more docs and tests, add support for:
//  - pointer fields
//  - subsections
//  - multi-value variables (+ internal representation)
//  - returning error context (+ numeric error codes ?)
//  - multiple readers (strings, files)
//  - escaping in strings and long(er) lines (?) (+ regexp-free parser)
//  - modifying files
//  - exporting files (+ metadata handling) (?)
//  - declare encoding (?)
//
package gcfg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
)

var (
	reCmnt    = regexp.MustCompile(`^([^;#"]*)[;#].*$`)
	reCmntQ   = regexp.MustCompile(`^([^;#"]*"[^"]*"[^;#"]*)[;#].*$`)
	reBlank   = regexp.MustCompile(`^\s*$`)
	reSect    = regexp.MustCompile(`^\s*\[\s*([^\s]*)\s*\]\s*$`)
	reVar     = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*([^"\s]*)\s*$`)
	reVarQ    = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*"([^"\n\\]*)"\s*$`)
	reVarDflt = regexp.MustCompile(`^\s*\b(.*)\b\s*$`)
)

const (
	// Default value in case a value for a variable isn't provided.
	DefaultValue = "true"
)

type Bool bool

var boolValues = map[string]interface{}{
	"true": true, "yes": true, "on": true, "1": true,
	"false": false, "no": false, "off": false, "0": false}

func scan(state fmt.ScanState, values map[string]interface{}) (interface{}, error) {
	var rd []rune
	var r rune
	var err error
	for r, _, err = state.ReadRune(); err == nil; r, _, err = state.ReadRune() {
		rd = append(rd, r)
		part := false
		for s, v := range values {
			if strings.EqualFold(string(rd), s) {
				return v, err
			}
			if len(rd) < len(s) && strings.EqualFold(string(rd), s[:len(rd)]) {
				part = true
			}
		}
		if part == false {
			state.UnreadRune()
			return nil, errors.New("unsupported value " + string(rd))
		}
	}
	return nil, err
}

func (b *Bool) Scan(state fmt.ScanState, verb rune) error {
	v, err := scan(state, boolValues)
	switch bb := v.(type) {
	case bool:
		*b = Bool(bb)
		return err
	}
	return err
}

func unref(v reflect.Value) reflect.Value {
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func fieldFold(v reflect.Value, name string) reflect.Value {
	n := strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(n, fieldName)
	})
}

func set(cfg interface{}, sect, name, value string) error {
	vDest := unref(reflect.ValueOf(cfg))
	vSect := unref(fieldFold(vDest, sect))
	vName := unref(fieldFold(vSect, name))
	vAddr := vName.Addr().Interface()
	switch v := vAddr.(type) {
	case *string:
		*v = value
		return nil
	case *bool:
		vAddr = (*Bool)(v)
	}
	// attempt to read an extra rune to make sure the value is consumed 
	var r rune
	n, err := fmt.Sscanf(value, "%v%c", vAddr, &r)
	switch {
	case n < 1 || n == 1 && err != io.EOF:
		return fmt.Errorf("failed to parse %q as %#v: parse error %v", value,
			vName.Type(), err)
	case n > 1:
		return fmt.Errorf("failed to parse %q as %#v: extra characters", value,
			vName.Type())
	case n == 1 && err == io.EOF:
		return nil
	}
	panic("never reached")
}

// Parse reads gcfg formatted data from reader and sets the values into the
// corresponding fields in config. Config must be a pointer to a struct.  
func Parse(config interface{}, reader io.Reader) error {
	r := bufio.NewReader(reader)
	sect := (*string)(nil)
	for line := 1; true; line++ {
		l, pre, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return err
		} else if pre {
			return errors.New("line too long")
		}
		// exclude comments
		if c := reCmnt.FindSubmatch(l); c != nil {
			l = c[1]
		} else if c := reCmntQ.FindSubmatch(l); c != nil {
			l = c[1]
		}
		if !reBlank.Match(l) {
			// "switch" based on line contents
			if sec := reSect.FindSubmatch(l); sec != nil {
				strsec := string(sec[1])
				sect = &strsec
			} else if v, vq, vd := reVar.FindSubmatch(l),
				reVarQ.FindSubmatch(l), reVarDflt.FindSubmatch(l); //
			v != nil || vq != nil || vd != nil {
				if sect == nil {
					return errors.New("no section")
				}
				var name, value string
				if v != nil {
					name, value = string(v[1]), string(v[2])
				} else if vq != nil {
					name, value = string(vq[1]), string(vq[2])
				} else { // vd != nil
					name, value = string(vd[1]), DefaultValue
				}
				err := set(config, *sect, name, value)
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

// ParseString reads gcfg formatted data from str and sets the values into the
// corresponding fields in cfg. It is a wrapper for Parse(config, reader).
func ParseString(config interface{}, str string) error {
	r := strings.NewReader(str)
	return Parse(config, r)
}

// ParseFile reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in cfg. It is a wrapper for
// Parse(config, reader).
func ParseFile(config interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return Parse(config, f)
}
