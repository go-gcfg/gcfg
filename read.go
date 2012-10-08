package gcfg

import (
	"bufio"
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

func fieldFold(v reflect.Value, name string) reflect.Value {
	n := strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(n, fieldName)
	})
}

func set(cfg interface{}, sect, sub, name, value string) error {
	vPCfg := reflect.ValueOf(cfg)
	if vPCfg.Kind() != reflect.Ptr || vPCfg.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("config must be a pointer to a struct"))
	}
	vCfg := vPCfg.Elem()
	vSect := fieldFold(vCfg, sect)
	if !vSect.IsValid() {
		return fmt.Errorf("invalid section: section %q", sect)
	}
	if vSect.Kind() == reflect.Map {
		vst := vSect.Type()
		if vst.Key().Kind() != reflect.String ||
			vst.Elem().Kind() != reflect.Ptr ||
			vst.Elem().Elem().Kind() != reflect.Struct {
			panic(fmt.Errorf("map field for section must have string keys and "+
				" pointer-to-struct values: section %q", sect))
		}
		if vSect.IsNil() {
			vSect.Set(reflect.MakeMap(vst))
		}
		k := reflect.ValueOf(sub)
		pv := vSect.MapIndex(k)
		if !pv.IsValid() {
			vType := vSect.Type().Elem().Elem()
			pv = reflect.New(vType)
			vSect.SetMapIndex(k, pv)
		}
		vSect = pv.Elem()
	} else if vSect.Kind() != reflect.Struct {
		panic(fmt.Errorf("field for section must be a map or a struct: "+
			"section %q", sect))
	} else if sub != "" {
		return fmt.Errorf("invalid subsection: "+
			"section %q subsection %q", sect, sub)
	}
	vName := fieldFold(vSect, name)
	if !vName.IsValid() {
		return fmt.Errorf("invalid variable: "+
			"section %q subsection %q variable %q", sect, sub, name)
	}
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
	sect := (*string)(nil)
	sectsub := ""
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
			if sec := reSect.FindSubmatch(l); sec != nil {
				strsec := string(sec[1])
				sect, sectsub = &strsec, ""
			} else if sec := reSectSub.FindSubmatch(l); sec != nil {
				strsec := string(sec[1])
				strsub := string(sec[2])
				if strsub == "" {
					return fmt.Errorf("subsection name \"\" not allowed; " +
						"use [section-name] for blank subsection name")
				}
				sect, sectsub = &strsec, strsub
			} else if v, vq, vd := reVar.FindSubmatch(l),
				reVarQ.FindSubmatch(l), reVarDflt.FindSubmatch(l); //
			v != nil || vq != nil || vd != nil {
				if sect == nil {
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
