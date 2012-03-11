package tnetstring

import (
	"errors"
	"reflect"
	"strconv"
)

func Marshal(v interface{}) (s string, err error) {
	defer func() {
		if s, ok := recover().(string); ok {
			err = errors.New(s)
		}
	}()
	val := reflect.ValueOf(v)
	var b outbuf
	b.n = 20
	b.buf = make([]byte, b.n)
	lookupEncode(val.Kind())(&b, val)
	return string(b.buf[b.n:]), nil
}

var encodeTable []func(*outbuf, reflect.Value)

func init() {
	encodeTable = []func(*outbuf, reflect.Value){
		reflect.Invalid:   encodeNull,
		reflect.Bool:      encodeBool,
		reflect.Int:       encodeInt,
		reflect.Int8:      encodeInt,
		reflect.Int16:     encodeInt,
		reflect.Int32:     encodeInt,
		reflect.Int64:     encodeInt,
		reflect.Uint:      encodeUint,
		reflect.Uint8:     encodeUint,
		reflect.Uint16:    encodeUint,
		reflect.Uint32:    encodeUint,
		reflect.Uint64:    encodeUint,
		reflect.Uintptr:   encodeUint,
		reflect.String:    encodeString,
		reflect.Array:     encodeArray,
		reflect.Slice:     encodeArray,
		reflect.Map:       encodeMap,
		reflect.Struct:    encodeStruct,
		reflect.Ptr:       encodeIndirect,
		reflect.Interface: encodeIndirect,
		// include last item so the array has the right length
		reflect.UnsafePointer: nil,
	}
}

func encodeNull(b *outbuf, v reflect.Value) {
	b.writeTString('~', "")
}

func encodeBool(b *outbuf, v reflect.Value) {
	str := "false"
	if v.Bool() {
		str = "true"
	}
	b.writeTString('!', str)
}

func encodeInt(b *outbuf, v reflect.Value) {
	b.writeTString('#', strconv.FormatInt(v.Int(), 10))
}

func encodeUint(b *outbuf, v reflect.Value) {
	b.writeTString('#', strconv.FormatUint(v.Uint(), 10))
}

func encodeString(b *outbuf, v reflect.Value) {
	b.writeTString(',', v.String())
}

func encodeArray(b *outbuf, v reflect.Value) {
	mark := b.mark(']')
	encodeFunc := lookupEncode(v.Type().Elem().Kind())
	for i := v.Len() - 1; i >= 0; i-- {
		encodeFunc(b, v.Index(i))
	}
	b.writeLen(mark)
}

func encodeMap(b *outbuf, v reflect.Value) {
	mark := b.mark('}')
	mapType := v.Type()
	if mapType.Key().Kind() != reflect.String {
		panic("tnetstring: only maps with string keys can be encoded")
	}
	encodeFunc := lookupEncode(mapType.Elem().Kind())
	for _, key := range v.MapKeys() {
		encodeFunc(b, v.MapIndex(key))
		b.writeTString(',', key.String())
	}
	b.writeLen(mark)
}

func encodeStruct(b *outbuf, v reflect.Value) {
	mark := b.mark('}')
	t := v.Type()
	l := t.NumField()
	for i := l - 1; i >= 0; i-- {
		field := t.Field(i)
		str := field.Tag.Get("tnetstring")
		if str == "" {
			str = field.Name
		}
		lookupEncode(field.Type.Kind())(b, v.Field(i))
		b.writeTString(',', str)
	}
	b.writeLen(mark)
}

func encodeIndirect(b *outbuf, v reflect.Value) {
	for {
		switch kind := v.Kind(); kind {
		case reflect.Ptr, reflect.Interface:
			v = v.Elem()
		default:
			lookupEncode(kind)(b, v)
			return
		}
	}
}

func lookupEncode(k reflect.Kind) func(*outbuf, reflect.Value) {
	if f := encodeTable[k]; f != nil {
		return f
	}
	panic("tnetstring: unsupported type")
}

type outbuf struct {
	buf []byte
	// n is the index we last wrote to or the amount of space left,
	// depending on how you look at it
	n int
}

func (buf *outbuf) writeByte(b byte) {
	if buf.n <= 0 {
		buf.grow(1)
	}
	buf.n--
	buf.buf[buf.n] = b
}

func (b *outbuf) mark(typ byte) int {
	b.writeByte(typ)
	return len(b.buf) - b.n
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

func (b *outbuf) writeLen(mark int) {
	l := len(b.buf) - b.n - mark
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
