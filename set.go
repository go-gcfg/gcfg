package gcfg

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// Implicit value string in case a value for a variable isn't provided.
	implicitValue = "true"
)

func fieldFold(v reflect.Value, name string) reflect.Value {
	var n string
	r0, _ := utf8.DecodeRuneInString(name)
	if unicode.IsLetter(r0) && !unicode.IsLower(r0) && !unicode.IsUpper(r0) {
		n = "X"
	}
	n += strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		return v.FieldByName(fieldName).CanSet() &&
			strings.EqualFold(n, fieldName)
	})
}

type setter func(destp interface{}, val string) error

var setterUnsupportedType = fmt.Errorf("unsupported type")

var setters = []setter{
	stringSetter, boolSetter, textUnmarshalerSetter, scanSetter,
}

func stringSetter(d interface{}, val string) error {
	dsp, ok := d.(*string)
	if !ok {
		return setterUnsupportedType
	}
	*dsp = val
	return nil
}

func textUnmarshalerSetter(d interface{}, val string) error {
	dtu, ok := d.(textUnmarshaler)
	if !ok {
		return setterUnsupportedType
	}
	return dtu.UnmarshalText([]byte(val))
}

func boolSetter(d interface{}, val string) error {
	dbp, ok := d.(*bool)
	if !ok {
		return setterUnsupportedType
	}
	return (*gbool)(dbp).UnmarshalText([]byte(val))
}

func scanSetter(d interface{}, val string) error {
	t := reflect.ValueOf(d).Elem().Type()
	verb := scanverb(t)
	// attempt to read an extra rune to make sure the value is consumed
	var r rune
	n, err := fmt.Sscanf(val, "%"+string(verb)+"%c", d, &r)
	switch {
	case n < 1 || n == 1 && err != io.EOF:
		return fmt.Errorf("failed to parse %q as %v: %v", val, t, err)
	case n > 1:
		return fmt.Errorf("failed to parse %q as %v: extra characters", val, t)
	}
	// n == 1 && err == io.EOF
	return nil
}

var typeVerbs = map[reflect.Type]rune{
	reflect.TypeOf(int(0)):    'd',
	reflect.TypeOf(int8(0)):   'd',
	reflect.TypeOf(int16(0)):  'd',
	reflect.TypeOf(int32(0)):  'd',
	reflect.TypeOf(int64(0)):  'd',
	reflect.TypeOf(uint(0)):   'd',
	reflect.TypeOf(uint8(0)):  'd',
	reflect.TypeOf(uint16(0)): 'd',
	reflect.TypeOf(uint32(0)): 'd',
	reflect.TypeOf(uint64(0)): 'd',
}

func scanverb(t reflect.Type) rune {
	verb, ok := typeVerbs[t]
	if !ok {
		return 'v'
	}
	return verb
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
	var vAddr reflect.Value
	// multi-value if unnamed slice type
	isMulti := vName.Type().Name() == "" && vName.Kind() == reflect.Slice
	if isMulti {
		// create new value and append to slice later
		vAddr = reflect.New(vName.Type().Elem())
	} else {
		vAddr = vName.Addr()
	}
	vAddrI := vAddr.Interface()
	err, ok := error(nil), false
	for _, s := range setters {
		err = s(vAddrI, value)
		if err == nil {
			ok = true
			break
		}
		if err != setterUnsupportedType {
			return err
		}
	}
	if !ok {
		// in case all setters returned setterUnsupportedType
		return err
	}
	if isMulti {
		vName.Set(reflect.Append(vName, vAddr.Elem()))
	}
	return nil
}
