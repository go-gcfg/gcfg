// Package gcfg reads "gitconfig-like" text-based configuration files with
// "name=value" pairs grouped into sections (gcfg files).
// Support for writing gcfg files may be added later.
//
// See ReadInto and the examples to get an idea of how to use it.
//
// This package is still a work in progress, and both the supported syntax and
// the API is subject to change.
//
// The syntax is based on that used by git config:
// http://git-scm.com/docs/git-config#_syntax .
// Note that the gcfg syntax may diverge from that of git config in the future
// to a limited degree.
// Currently planned differences (apart from TODOs listed below) are:
//  - gcfg files must use UTF-8 encoding and must not contain the 0 byte
//  - include and "path" type is not supported at the package level
//  - `[sec.sub]` format is not allowed (deprecated in gitconfig)
//  - `[sec ""]` is not allowed
//    - `[sec]` is the equivalent (section name "sec" and empty subsection name)
//  - within a single file, definitions must be consecutive for each:
//    - section: '[secA]' -> '[secB]' -> '[secA]' is an error
//    - subsection: '[sec "A"]' -> '[sec "B"]' -> '[sec "A"]' is an error
//    - multivalued variable: 'multi=a' -> 'other=x' -> 'multi=b' is an error
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
//    - support partially quoted strings
//    - support escaping in strings
//    - support multiple inputs (readers, strings, files)
//    - support declaring encoding (?)
//    - support pointer fields
//  - ScanEnum
//    - should use longest match (?)
//    - support matching on unique prefix (?)
//  - writing gcfg files
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
	defaultValue = "true"
)

type gbool bool

var gboolValues = map[string]interface{}{
	"true": true, "yes": true, "on": true, "1": true,
	"false": false, "no": false, "off": false, "0": false}

func (b *gbool) Scan(state fmt.ScanState, verb rune) error {
	v, err := ScanEnum(state, gboolValues, true)
	if err != nil {
		return err
	}
	bb, _ := v.(bool) // cannot be non-bool
	*b = gbool(bb)
	return nil
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

// ReadInto reads gcfg formatted data from reader and sets the values into the
// corresponding fields in config.
//
// Config must be a pointer to a struct.
// Each section corresponds to a struct field in config, and each variable in a
// section corresponds to a data field in the section struct.
// The name of the field must match the name of the section or variable,
// ignoring case.
// Hyphens in variable names correspond to underscores in section or field
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
// For all other types, fmt.Sscanf is used to parse the value and set it to the
// field.
// This means that built-in Go types are parseable using the standard format,
// and any user-defined type is parseable if it implements the fmt.Scanner
// interface.
//
// See ReadStringInto for examples.
//
func ReadInto(config interface{}, reader io.Reader) error {
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
					name, value = string(vd[1]), defaultValue
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

// ReadStringInto reads gcfg formatted data from str and sets the values into
// the corresponding fields in config.
// See ReadInto(config, reader) for a detailed description of how values are
// parsed and set into config.
func ReadStringInto(config interface{}, str string) error {
	r := strings.NewReader(str)
	return ReadInto(config, r)
}

// ReadFileInto reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in config.
// See ReadInto(config, reader) for a detailed description of how values are
// parsed and set into config.
func ReadFileInto(config interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return ReadInto(config, f)
}
