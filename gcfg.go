// Package gcfg reads "gitconfig-like" text-based configuration files with
// "name=value" pairs grouped into sections (gcfg files).
// Support for modifying and/or exporting gcfg files may be added later.
//
// See Parse and the examples to get an idea of how to use it.
//
// This package is still a work in progress, and both the supported syntax and
// the API is subject to change.
//
// The syntax is based on that used by git config:
// http://git-scm.com/docs/git-config#_syntax .
// Note that the gcfg syntax may diverge from that of git config in the future
// to a limited degree.
// Current differences (apart from TODOs listed below) are:
//  - gcfg files must use UTF-8 encoding (for now)
//  - include and "path" type is not supported at the package level
//  - `[sec.sub]` format is not allowed (deprecated in gitconfig)
//  - `[sec ""]` is not allowed
//    - `[sec]` is the equivalent (section name "sec" and empty subsection name)
//
// The package may be usable for handling some of the various "INI file" formats
// used by some programs and libraries, but achieving or maintaining
// compatibility with any of those is not a primary concern.
//
// TODO:
//  - docs
//    - format spec
//  - parsing
//    - define internal representation structure
//    - support multi-value variables
//    - non-regexp based parser
//    - support escaping in strings
//    - support multiple inputs (readers, strings, files)
//    - support declaring encoding (?)
//    - support pointer fields
//  - modifying / exporting gcfg files
//  - error handling
//    - include error context
//    - more helpful error messages
//    - error types / codes?
//    - limit input size?
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
	reSect    = regexp.MustCompile(`^\s*\[\s*([^"\s]*)\s*\]\s*$`)
	reSectSub = regexp.MustCompile(`^\s*\[\s*([^"\s]*)\s*"([^"]+)"\s*\]\s*$`)
	reVar     = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*([^"\s]*)\s*$`)
	reVarQ    = regexp.MustCompile(`^\s*([^"=\s]+)\s*=\s*"([^"\n\\]*)"\s*$`)
	reVarDflt = regexp.MustCompile(`^\s*\b(.*)\b\s*$`)
)

const (
	// Default value string in case a value for a variable isn't provided.
	DefaultValue = "true"
)

type gbool bool

var boolValues = map[string]interface{}{
	"true": true, "yes": true, "on": true, "1": true,
	"false": false, "no": false, "off": false, "0": false}

const (
	// initial size of buffer for ScanEnum
	scanEnumBufferHint = 16
)

func strEq(s1, s2 string, fold bool) bool {
	return fold && strings.EqualFold(s1, s2) || !fold && s1 == s2
}

// ScanEnum is a helper function to simplify the implementation of fmt.Scanner
// methods for "enum-like" types, that is, user-defined types where the set of
// values and string representations is fixed.
//
// State is the state passed to the implementation of the fmt.Scanner method.
// Values holds as map values the values of the type, with their string
// representations as keys.
// If fold is true, comparison of the string representation uses
// strings.EqualFold, otherwise the equal operator for strings.
//
// On a match, ScanEnum stops after reading the last rune of the matched string,
// and returns the corresponding value together with a nil error.
// On no match, ScanEnum attempts to unread the last rune (the first rune that
// could not potentially match any of the values), and returns a non-nil error,
// togetherwith a nil value for interface{}.
// On I/O error, ScanEnum returns the I/O error, together with a nil value for
// interface{}.
//
func ScanEnum(state fmt.ScanState, values map[string]interface{}, fold bool) (
	interface{}, error) {
	//
	rd := make([]rune, 0, scanEnumBufferHint)
	keys := make(map[string]struct{}, len(values)) // potential keys
	for s, _ := range values {
		keys[s] = struct{}{}
	}
	for {
		r, _, err := state.ReadRune()
		if err != nil {
			return nil, err
		}
		rd = append(rd, r)
		srd := string(rd)
		lrd := len(srd)
		for s, _ := range keys {
			if strEq(srd, s, fold) {
				return values[s], nil
			}
			if len(rd) < len(s) && !strEq(srd, s[:lrd], fold) {
				delete(keys, s)
			}
		}
		if len(keys) == 0 {
			state.UnreadRune()
			return nil, fmt.Errorf("unsupported value %q", srd)
		}
	}
	panic("never reached")
}

func (b *gbool) Scan(state fmt.ScanState, verb rune) error {
	v, err := ScanEnum(state, boolValues, true)
	switch bb := v.(type) {
	case bool:
		*b = gbool(bb)
		return err
	}
	return err
}

func fieldFold(v reflect.Value, name string) reflect.Value {
	n := strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(n, fieldName)
	})
}

func set(cfg interface{}, sect, sub, name, value string) error {
	vDest := reflect.ValueOf(cfg).Elem()
	vSect := fieldFold(vDest, sect)
	if vSect.Kind() == reflect.Map {
		if vSect.IsNil() {
			vSect.Set(reflect.MakeMap(vSect.Type()))
		}
		k := reflect.ValueOf(sub)
		pv := vSect.MapIndex(k)
		if !pv.IsValid() {
			vType := vSect.Type().Elem().Elem()
			pv = reflect.New(vType)
			vSect.SetMapIndex(k, pv)
		}
		vSect = pv.Elem()
	} else if sub != "" {
		return fmt.Errorf("expected map; section %q subsection %q", sect, sub)
	}
	vName := fieldFold(vSect, name)
	vAddr := vName.Addr().Interface()
	switch v := vAddr.(type) {
	case *string:
		*v = value
		return nil
	case *bool:
		vAddr = (*gbool)(v)
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
// corresponding fields in config.
//
// Config must be a pointer to a struct.
// Each section corresponds to a struct field in config, and each variable in a
// section corresponds to a data field in the section struct.
// The field names must match that of the section or the variable, ignoring
// case.
// Hyphens are replaced with underscores.
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
// For all other types, fmt.Sscanf is used to parse the value and set it to the
// field.
// This means that built-in Go types are parseable using the standard format,
// and any user-defined types is parseable if it implements the fmt.Scanner
// interface.
//
// See ParseString for examples.
//
func Parse(config interface{}, reader io.Reader) error {
	r := bufio.NewReader(reader)
	sect := (*string)(nil)
	sectsub := ""
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
				sect, sectsub = &strsec, ""
			} else if sec := reSectSub.FindSubmatch(l); sec != nil {
				strsec := string(sec[1])
				strsub := string(sec[2])
				if strsub == "" {
					return errors.New("empty subsection not allowed")
				}
				sect, sectsub = &strsec, strsub
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
				err := set(config, *sect, sectsub, name, value)
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
// corresponding fields in config.
// See Parse(config, reader) for the details of the parsing rules.
func ParseString(config interface{}, str string) error {
	r := strings.NewReader(str)
	return Parse(config, r)
}

// ParseFile reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in config.
// See Parse(config, reader) for the details of the parsing rules.
func ParseFile(config interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return Parse(config, f)
}
