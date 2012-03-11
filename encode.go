package tnetstring

import (
	"errors"
	"math"
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
	b.writeRawString("0:~")
}

func encodeBool(b *outbuf, v reflect.Value) {
	if v.Bool() {
		b.writeRawString("4:true!")
		return
	}
	b.writeRawString("5:false!")
}

func encodeInt(b *outbuf, v reflect.Value) {
	val := v.Int()
	var vallen int
	llen := 1
	switch {
	case val < -999999999:
		llen = 3
		fallthrough
	case val < 0:
		vallen = digitCount(uint64(-val)) + 1
	case val > 999999999:
		llen = 2
		fallthrough
	default:
		vallen = digitCount(uint64(val))
	}
	n := llen + 1 + vallen + 1
	if b.n < n {
		b.grow(n)
	}
	b.n--
	b.buf[b.n] = '#'
	b.n -= vallen
	valstr := strconv.AppendInt(b.buf[b.n:b.n], val, 10)
	l := len(valstr)
	b.n--
	b.buf[b.n] = ':'
	b.n -= llen
	strconv.AppendUint(b.buf[:b.n], uint64(l), 10)
}

func encodeUint(b *outbuf, v reflect.Value) {
	val := v.Uint()
	vallen := digitCount(val)
	llen := 1
	if val > 999999999 {
		llen = 2
	}
	n := llen + 1 + vallen + 1
	if b.n < n {
		b.grow(n)
	}
	b.n--
	b.buf[b.n] = '#'
	b.n -= vallen
	valstr := strconv.AppendUint(b.buf[b.n:b.n], val, 10)
	l := len(valstr)
	b.n--
	b.buf[b.n] = ':'
	b.n -= llen
	strconv.AppendUint(b.buf[:b.n], uint64(l), 10)
}

func encodeString(b *outbuf, v reflect.Value) {
	b.writeString(v.String())
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
		b.writeString(key.String())
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
		b.writeString(str)
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

func (b *outbuf) writeRawString(s string) {
	n := len(s)
	if b.n < n {
		b.grow(n)
	}
	b.n -= n
	copy(b.buf[b.n:], s)
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

func (b *outbuf) writeString(s string) {
	l := len(s)
	llen := digitCount(uint64(l))
	n := llen + 1 + l + 1
	if b.n < n {
		b.grow(n)
	}
	b.n--
	b.buf[b.n] = ','
	b.n -= l
	copy(b.buf[b.n:], s)
	b.n--
	b.buf[b.n] = ':'
	b.n -= llen
	strconv.AppendInt(b.buf[:b.n], int64(l), 10)
}

func (b *outbuf) writeLen(mark int) {
	l := uint64(len(b.buf) - b.n - mark)
	b.writeByte(':')
	llen := digitCount(l)
	if b.n < llen {
		b.grow(llen)
	}
	b.n -= llen
	strconv.AppendUint(b.buf[:b.n], l, 10)
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

func digitCount(n uint64) int {
	if n == 0 {
		return 1
	}
	return int(math.Log10(float64(n)))+1
}
