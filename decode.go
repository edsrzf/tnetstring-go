package tnetstring

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

func Unmarshal(data string, v interface{}) error {
	val := reflect.ValueOf(v)
	val = reflect.Indirect(val)
	if !val.CanSet() {
		return errors.New("tnetstring: Unmarshal requires a settable value")
	}
	_, err := unmarshal(data, val)
	return err
}

func indirect(v reflect.Value) reflect.Value {
	for {
		switch v.Kind() {
		case reflect.Ptr:
			print("Ptr")
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		case reflect.Interface:
			print("Interface")
			if v.IsNil() {
				return v
			}
			v = v.Elem()
		default:
			print("Default")
			return v
		}
	}
	panic("unreachable")
}

var typeLookup = [...]byte{
	reflect.Invalid: '~',
	reflect.Bool:    '!',
	reflect.Int:     '#',
	reflect.Int8:    '#',
	reflect.Int16:   '#',
	reflect.Int32:   '#',
	reflect.Int64:   '#',
	reflect.Uint:    '#',
	reflect.Uint8:   '#',
	reflect.Uint16:  '#',
	reflect.Uint32:  '#',
	reflect.Uint64:  '#',
	reflect.Uintptr: '#',
	reflect.Float32: '^',
	reflect.Float64: '^',
	reflect.String:  ',',
	reflect.Array:   ']',
	reflect.Slice:   ']',
	reflect.Map:     '}',
	reflect.Struct:  '}',
	// include last item so the array has the right length
	reflect.UnsafePointer: 0,
}

func unmarshal(data string, v reflect.Value) (int, error) {
	typ, content, n := readElement(data)
	if n == 0 {
		return 0, errors.New("tnetstring: invalid data")
	}
	v = indirect(v)
	kind := v.Kind()
	// ~ and interface types are special cases
	if typ != '~' && kind != reflect.Interface && typeLookup[kind] != typ {
		return 0, errors.New("tnetstring: invalid value to unmarshal into")
	}
	switch typ {
	case '!':
		b := content == "true"
		if kind == reflect.Bool {
			v.SetBool(b)
		} else {
			v.Set(reflect.ValueOf(b))
		}
	case '#':
		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64:
			i, err := strconv.ParseInt(content, 10, 64)
			if err != nil {
				return 0, err
			}
			v.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			ui, err := strconv.ParseUint(content, 10, 64)
			if err != nil {
				return 0, err
			}
			v.SetUint(ui)
		case reflect.Interface:
			i, err := strconv.ParseInt(content, 10, 64)
			if err != nil {
				return 0, err
			}
			v.Set(reflect.ValueOf(i))
		}
	case '^':
		f, err := strconv.ParseFloat(content, 64)
		if err != nil {
			return 0, err
		}
		switch kind {
		case reflect.Float32, reflect.Float64:
			v.SetFloat(f)
		case reflect.Interface:
			v.Set(reflect.ValueOf(f))
		}
	case ',':
		if kind == reflect.String {
			v.SetString(content)
		} else {
			v.Set(reflect.ValueOf(content))
		}
	case ']':
		unmarshalArray(content, v, kind)
	case '}':
		var err error
		if kind == reflect.Map {
			err = unmarshalMap(content, v)
		} else {
			err = unmarshalStruct(content, v)
		}
		if err != nil {
			return 0, err
		}
	case '~':
		switch kind {
		case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			v.Set(reflect.Zero(v.Type()))
		default:
			return 0, errors.New("tnetstring: invalid value to unmarshal into")
		}
	default:
		return 0, errors.New("tnetstring: unknown type")
	}
	return n, nil
}

func unmarshalArray(data string, v reflect.Value, kind reflect.Kind) error {
	i := 0
	vtype := v.Type()
	l := v.Cap()
	if v.Len() < l {
		v.SetLen(l)
	}
	for len(data) > 0 {
		if i >= l {
			if kind == reflect.Array {
				break
			}
			newl := l + l/2
			if newl < 4 {
				newl = 4
			}
			newv := reflect.MakeSlice(vtype, newl, newl)
			reflect.Copy(newv, v)
			v.Set(newv)
			l = newl
		}
		el := v.Index(i)
		i++
		n, err := unmarshal(data, el)
		data = data[n:]
		if err != nil {
			return err
		}
	}
	if i < l {
		if kind == reflect.Array {
			// zero out the rest
			z := reflect.Zero(vtype.Elem())
			for ; i < l; i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(i)
		}
	}
	return nil
}

func unmarshalMap(data string, v reflect.Value) error {
	mapType := v.Type()
	if mapType.Key().Kind() != reflect.String {
		return errors.New("tnetstring: only maps with string keys can be unmarshaled")
	}
	if v.IsNil() {
		v.Set(reflect.MakeMap(mapType))
	}
	vtype := mapType.Elem()
	var s string
	key := reflect.ValueOf(&s).Elem()
	for len(data) > 0 {
		val := reflect.New(vtype).Elem()
		typ, content, n := readElement(data)
		data = data[n:]
		if typ != ',' {
			return errors.New("tnetstring: non-string key in dictionary")
		}
		s = content
		n, err := unmarshal(data, val)
		data = data[n:]
		if err != nil {
			return err
		}
		v.SetMapIndex(key, val)
	}
	return nil
}

func unmarshalStruct(data string, v reflect.Value) error {
	structType := v.Type()
	var name string
	for len(data) > 0 {
		typ, content, n := readElement(data)
		data = data[n:]
		if typ != ',' {
			return errors.New("tnetstring: non-string key in dictionary")
		}
		name = content
		field := v.FieldByName(name)
		if !field.IsValid() {
			for i := 0; i < structType.NumField(); i++ {
				f := structType.Field(i)
				if f.Tag.Get("tnetstring") == name {
					field = v.Field(i)
					break
				}
			}
			if !field.IsValid() {
				// skip the field
				_, _, n := readElement(data)
				data = data[n:]
				continue
			}
		}
		n, err := unmarshal(data, field)
		data = data[n:]
		if err != nil {
			return err
		}
	}
	return nil
}

func readElement(data string) (typ byte, content string, n int) {
	col := strings.IndexRune(data, ':')
	if col < 1 {
		return
	}
	n, err := strconv.Atoi(data[:col])
	// use the position after the colon from here on out
	col++
	if err != nil || col+n > len(data) {
		return
	}
	n += col
	content = data[col:n]
	typ = data[n]
	n++
	return
}
