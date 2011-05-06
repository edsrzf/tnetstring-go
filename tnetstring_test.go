package tnetstring

import (
	"reflect"
	"testing"
)

var tnetstringTests = []struct {
	val  interface{}
	data string
}{
	{nil, "0:~"},
	{true, "4:true!"},
	{false, "5:false!"},
	{14, "2:14#"},
	{uint(14), "2:14#"},
	{"hello", "5:hello,"},
	{[]bool{}, "0:]"},
	{[]int{1, 2, 3}, "12:1:1#1:2#1:3#]"},
	{[...]int{1, 2, 3}, "12:1:1#1:2#1:3#]"},
	{[]string{"ab", "cd", "ef"}, "15:2:ab,2:cd,2:ef,]"},
	{map[string]string{}, "0:}"},
	// can't test more than one map element due to undefined order
	{map[string]int{"a": 1}, "8:1:a,1:1#}"},
	{struct {
		A int
		B string
	}{1, "hello"}, "20:1:A,1:1#1:B,5:hello,}"},
	{&struct {
		A int
		B string
	}{1, "hello"}, "20:1:A,1:1#1:B,5:hello,}"},
}

func TestMarshal(t *testing.T) {
	for i, test := range tnetstringTests {
		b, err := Marshal(test.val)
		if err != nil {
			t.Errorf("#%d Marshal error: %s", i, err)
			continue
		}
		if string(b) != test.data {
			t.Errorf("#%d want\n%q\ngot\n%q", i, test.data, b)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	for i, test := range tnetstringTests {
		ty := reflect.TypeOf(test.val)
		if ty == nil {
			continue
		}
		val := reflect.New(ty)
		err := Unmarshal([]byte(test.data), val.Interface())
		if err != nil {
			t.Errorf("#%d Unmarshal error: %s", i, err)
		}
		if !reflect.DeepEqual(test.val, val.Elem().Interface()) {
			t.Errorf("#%d want\n%v\ngot\n%v", i, test.val, val.Elem().Interface())
		}
	}
}
