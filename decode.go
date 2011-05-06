package tnetstring

import (
	"bytes"
	"os"
	"reflect"
	"strconv"
)

func Unmarshal(data []byte, v interface{}) os.Error {
	val := reflect.ValueOf(v)
	val = reflect.Indirect(val)
	if !val.CanSet() {
		return os.NewError("tnetstring: Unmarshal requires a settable value")
	}
	_, err := unmarshal(data, val)
	return err
}

func indirect(v reflect.Value) reflect.Value {
	for {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			fallthrough
		case reflect.Interface:
			v = v.Elem()
		default:
			return v
		}
	}
	panic("unreachable")
}

func unmarshal(data []byte, v reflect.Value) (int, os.Error) {
	typ, content, n := readElement(data)
	if n == 0 {
		return 0, os.NewError("tnetstring: invalid data")
	}
	v = indirect(v)
	kind := v.Kind()
	if typeLookup[kind] != typ {
		return 0, os.NewError("tnetstring: invalid value to unmarshal into")
	}
	switch typ {
	case '!':
		v.SetBool(string(content) == "true")
	case '#':
		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.Atoi64(string(content))
			if err != nil {
				return 0, err
			}
			v.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			ui, err := strconv.Atoui64(string(content))
			if err != nil {
				return 0, err
			}
			v.SetUint(ui)
		}
	case ',':
		v.SetString(string(content))
	case ']':
		unmarshalArray(content, v)
	case '}':
		var err os.Error
		if kind == reflect.Map {
			err = unmarshalMap(content, v)
		} else {
			err = unmarshalStruct(content, v)
		}
		if err != nil {
			return 0, err
		}
	default:
		return 0, os.NewError("tnetstring: unknown type")
	}
	return n, nil
}

func unmarshalArray(data []byte, v reflect.Value) os.Error {
	kind := v.Kind()
	n := 0
	i := 0
	elType := v.Type().Elem()
	for len(data)-n > 0 {
		if i >= v.Len() {
			if kind == reflect.Array {
				break
			} else {
				// TODO: could cut down on allocations by calling MakeSlice instead
				v.Set(reflect.Append(v, reflect.New(elType).Elem()))
			}
		}
		el := v.Index(i)
		i++
		nn, err := unmarshal(data[n:], el)
		if err != nil {
			return err
		}
		n += nn
	}
	return nil
}

func unmarshalMap(data []byte, v reflect.Value) os.Error {
	ktype := v.Type().Key()
	if ktype.Kind() != reflect.String {
		return os.NewError("tnetstring: only maps with string keys can be unmarshaled")
	}
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	n := 0
	vtype := v.Type().Elem()
	key := reflect.New(ktype).Elem()
	val := reflect.New(vtype).Elem()
	for len(data)-n > 0 {
		nn, err := unmarshal(data[n:], key)
		if err != nil {
			return err
		}
		n += nn
		nn, err = unmarshal(data[n:], val)
		if err != nil {
			return err
		}
		n += nn
		v.SetMapIndex(key, val)
	}
	return nil
}

func unmarshalStruct(data []byte, v reflect.Value) os.Error {
	n := 0
	structType := v.Type()
	var s string
	name := reflect.ValueOf(&s).Elem()
	for len(data)-n > 0 {
		nn, err := unmarshal(data[n:], name)
		if err != nil {
			return err
		}
		n += nn
		field := v.FieldByName(s)
		if field.Internal == nil {
			for i := 0; i < structType.NumField(); i++ {
				f := structType.Field(i)
				if f.Tag == s {
					field = v.Field(i)
					break
				}
			}
			if field.Internal == nil {
				var i interface{}
				field = reflect.ValueOf(&i).Elem()
			}
		}
		nn, err = unmarshal(data[n:], field)
		if err != nil {
			return err
		}
		n += nn
	}
	return nil
}

func readElement(data []byte) (typ byte, content []byte, n int) {
	col := bytes.IndexByte(data, ':')
	if col < 1 {
		return
	}
	n, err := strconv.Atoi(string(data[:col]))
	if err != nil || n > len(data[col+1:]) {
		return
	}
	// +1 for colon
	n += col + 1
	content = data[col+1 : n]
	typ = data[n]
	n++
	return
}
