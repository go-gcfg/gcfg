package gcfg

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"code.google.com/p/gcfg/types"
)

const (
	// Implicit value string in case a value for a variable isn't provided.
	implicitValue = "true"
)

type tag struct {
	ident string
}

func newTag(t string) tag {
	idx := strings.IndexRune(t, ',')
	if idx < 0 {
		idx = len(t)
	}
	id := t[0:idx]
	return tag{ident: id}
}

func fieldFold(v reflect.Value, name string) reflect.Value {
	var n string
	r0, _ := utf8.DecodeRuneInString(name)
	if unicode.IsLetter(r0) && !unicode.IsLower(r0) && !unicode.IsUpper(r0) {
		n = "X"
	}
	n += strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		if !v.FieldByName(fieldName).CanSet() {
			return false
		}
		f, _ := v.Type().FieldByName(fieldName)
		t := newTag(f.Tag.Get("gcfg"))
		if t.ident != "" {
			return strings.EqualFold(t.ident, name)
		}
		return strings.EqualFold(n, fieldName)
	})
}

type setter func(destp interface{}, val string) error

var errUnsupportedType = fmt.Errorf("unsupported type")

var setters = []setter{
	textUnmarshalerSetter, typeSetter, kindSetter, scanSetter,
}

func textUnmarshalerSetter(d interface{}, val string) error {
	dtu, ok := d.(textUnmarshaler)
	if !ok {
		return errUnsupportedType
	}
	return dtu.UnmarshalText([]byte(val))
}

func boolSetter(d interface{}, val string) error {
	b, err := types.ParseBool(val)
	if err == nil {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(b))
	}
	return err
}

func intSetterDecHex(d interface{}, val string) error {
	return types.ParseInt(d, val, types.Dec+types.Hex)
}

func stringSetter(d interface{}, val string) error {
	dsp, ok := d.(*string)
	if !ok {
		return errUnsupportedType
	}
	*dsp = val
	return nil
}

var kindSetters = map[reflect.Kind]setter{
	reflect.String: stringSetter,
	reflect.Bool:   boolSetter,
}

var typeSetters = map[reflect.Type]setter{
	reflect.TypeOf(int(0)):    intSetterDecHex,
	reflect.TypeOf(int8(0)):   intSetterDecHex,
	reflect.TypeOf(int16(0)):  intSetterDecHex,
	reflect.TypeOf(int32(0)):  intSetterDecHex,
	reflect.TypeOf(int64(0)):  intSetterDecHex,
	reflect.TypeOf(uint(0)):   intSetterDecHex,
	reflect.TypeOf(uint8(0)):  intSetterDecHex,
	reflect.TypeOf(uint16(0)): intSetterDecHex,
	reflect.TypeOf(uint32(0)): intSetterDecHex,
	reflect.TypeOf(uint64(0)): intSetterDecHex,
}

func typeSetter(d interface{}, val string) error {
	t := reflect.ValueOf(d).Elem().Type()
	setter, ok := typeSetters[t]
	if !ok {
		return errUnsupportedType
	}
	return setter(d, val)
}

func kindSetter(d interface{}, val string) error {
	k := reflect.ValueOf(d).Elem().Kind()
	setter, ok := kindSetters[k]
	if !ok {
		return errUnsupportedType
	}
	return setter(d, val)
}

func scanSetter(d interface{}, val string) error {
	t := reflect.ValueOf(d).Elem().Type()
	// attempt to read an extra rune to make sure the value is consumed
	var r rune
	n, err := fmt.Sscanf(val, "%v%c", d, &r)
	switch {
	case n < 1 || n == 1 && err != io.EOF:
		return fmt.Errorf("failed to parse %q as %v: %v", val, t, err)
	case n > 1:
		return fmt.Errorf("failed to parse %q as %v: extra characters", val, t)
	}
	// n == 1 && err == io.EOF
	return nil
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
		if err != errUnsupportedType {
			return err
		}
	}
	if !ok {
		// in case all setters returned errUnsupportedType
		return err
	}
	if isMulti {
		vName.Set(reflect.Append(vName, vAddr.Elem()))
	}
	return nil
}
