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
//  - comments
//  - quoted strings
//  - hyphens in section names
//  - git-compatible bools 
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
	reSect    = regexp.MustCompile(`^\b*\[(.*)\]\b*$`)
	reVar     = regexp.MustCompile(`^\b*(.*)\b*=\b*(.*)\b*$`)
	reVarDflt = regexp.MustCompile(`^\b*(.+)\b*$`)
)

const (
	DefaultValue = "true"
)

func unref(v reflect.Value) reflect.Value {
	for reflect.Ptr == v.Type().Kind() {
		v = v.Elem()
	}
	return v
}

func fieldFold(v reflect.Value, name string) reflect.Value {
	return v.FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(name, fieldName)
	})
}

func set(cfg interface{}, sect, name, value string) error {
	vDest := unref(reflect.ValueOf(cfg))
	vSect := unref(fieldFold(vDest, sect))
	vName := unref(fieldFold(vSect, name))
	fmt.Sscan(value, vName.Addr().Interface())
	return nil
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
		// "switch" based on line contents
		if sec := reSect.FindSubmatch(l); sec != nil {
			strsec := string(sec[1])
			sect = &strsec
		} else if v, vd := reVar.FindSubmatch(l), reVarDflt.FindSubmatch(l); v != nil || vd != nil {
			if sect == nil {
				return errors.New("no section")
			}
			var name, value string
			if v != nil {
				name, value = string(v[1]), string(v[2])
			} else { // vd != nil
				name, value = string(vd[1]), DefaultValue
			}
			set(config, *sect, name, value)
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
