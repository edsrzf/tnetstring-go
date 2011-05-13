package tnetstring

import (
	"os"
	"reflect"
	"strconv"
)

type outbuf struct {
	buf []byte
	// n is the index we last wrote to or the amount of space left,
	// depending on how you look at it
	n int
}

func Marshal(v interface{}) (string, os.Error) {
	val := reflect.ValueOf(v)
	b := new(outbuf)
	b.n = 20
	b.buf = make([]byte, b.n)
	if err := marshal(b, val); err != nil {
		return "", err
	}
	return string(b.buf[b.n:]), nil
}

func marshal(b *outbuf, v reflect.Value) os.Error {
	v = indirect(v, false)
	var typ byte
	var str string
	switch v.Kind() {
	case reflect.Invalid:
		typ = '~'
	case reflect.Bool:
		typ = '!'
		if v.Bool() {
			str = "true"
		} else {
			str = "false"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		typ = '#'
		str = strconv.Itoa64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		typ = '#'
		str = strconv.Uitoa64(v.Uint())
	case reflect.String:
		typ = ','
		str = v.String()
	case reflect.Array, reflect.Slice:
		b.writeByte(']')
		orig := len(b.buf) - b.n
		for i := v.Len() - 1; i >= 0; i-- {
			if err := marshal(b, v.Index(i)); err != nil {
				return err
			}
		}
		b.writeLen(orig)
		return nil
	case reflect.Map:
		b.writeByte('}')
		orig := len(b.buf) - b.n
		if v.Type().Key().Kind() != reflect.String {
			return os.NewError("tnetstring: only maps with string keys can be marshaled")
		}
		for _, key := range v.MapKeys() {
			if err := marshal(b, v.MapIndex(key)); err != nil {
				return err
			}
			b.writeTString(',', key.String())
		}
		b.writeLen(orig)
		return nil
	case reflect.Struct:
		b.writeByte('}')
		orig := len(b.buf) - b.n
		t := v.Type()
		l := t.NumField()
		for i := l - 1; i >= 0; i-- {
			field := t.Field(i)
			str := field.Tag
			if str == "" {
				str = field.Name
			}
			if err := marshal(b, v.Field(i)); err != nil {
				return err
			}
			b.writeTString(',', str)
		}
		b.writeLen(orig)
		return nil
	default:
		return os.NewError("tnetstring: unsupported type")
	}
	b.writeTString(typ, str)
	return nil
}

func (buf *outbuf) writeByte(b byte) {
	if buf.n <= 0 {
		buf.grow(1)
	}
	buf.n--
	buf.buf[buf.n] = b
}

func (b *outbuf) writeTString(typ byte, s string) {
	l := len(s)
	lstr := strconv.Itoa(l)
	n := len(lstr) + 1 + l + 1
	if b.n < n {
		b.grow(n)
	}
	b.n--
	b.buf[b.n] = typ
	b.n -= l
	copy(b.buf[b.n:], s)
	b.n--
	b.buf[b.n] = ':'
	b.n -= len(lstr)
	copy(b.buf[b.n:], lstr)
}

func (b *outbuf) writeLen(orig int) {
	l := len(b.buf) - b.n - orig
	str := strconv.Itoa(l)
	b.writeByte(':')
	if b.n < len(str) {
		b.grow(len(str))
	}
	b.n -= len(str)
	copy(b.buf[b.n:], str)
}

func (b *outbuf) grow(n int) {
	l := len(b.buf)
	need := 2 * l
	if need < l+n-b.n {
		need = l + n - b.n
	}
	buf := make([]byte, need)
	copy(buf[need-l+b.n:], b.buf[b.n:])
	b.buf = buf
	b.n = need - l + b.n
}
