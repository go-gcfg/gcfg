package gcfg

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

const (
	// Implicit value string in case a value for a variable isn't provided.
	implicitValue = "true"
)

func fieldFold(v reflect.Value, name string) reflect.Value {
	n := strings.Replace(name, "-", "_", -1)
	return v.FieldByNameFunc(func(fieldName string) bool {
		return strings.EqualFold(n, fieldName)
	})
}

var primitiveVerbs = map[reflect.Kind]rune{
	reflect.Int:    'd',
	reflect.Int8:   'd',
	reflect.Int16:  'd',
	reflect.Int32:  'd',
	reflect.Int64:  'd',
	reflect.Uint:   'd',
	reflect.Uint8:  'd',
	reflect.Uint16: 'd',
	reflect.Uint32: 'd',
	reflect.Uint64: 'd',
}

func scanverb(t reflect.Type) rune {
	if t.PkgPath() != "" {
		// user-defined type
		return 'v'
	}
	verb, ok := primitiveVerbs[t.Kind()]
	if !ok {
		verb = 'v'
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
	if v, ok := vAddrI.(*bool); ok {
		vAddrI = (*gbool)(v)
	}
	switch v := vAddrI.(type) {
	case *string:
		*v = value
	case textUnmarshaler:
		err := v.UnmarshalText([]byte(value))
		if err != nil {
			return err
		}
	default:
		// attempt to read an extra rune to make sure the value is consumed
		var r rune
		verb := scanverb(vAddr.Elem().Type())
		n, err := fmt.Sscanf(value, "%"+string(verb)+"%c", vAddrI, &r)
		switch {
		case n < 1 || n == 1 && err != io.EOF:
			return fmt.Errorf("failed to parse %q as %v: %v", value, vName.Type(),
				err)
		case n > 1:
			return fmt.Errorf("failed to parse %q as %v: extra characters", value,
				vName.Type())
		}
		// n == 1 && err == io.EOF
	}
	if isMulti {
		vName.Set(reflect.Append(vName, vAddr.Elem()))
	}
	return nil
}
