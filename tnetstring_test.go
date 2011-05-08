package tnetstring

import (
	"json"
	"os"
	"reflect"
	"testing"
)

type tnetstringTest struct {
	val  interface{}
	data string
}

// Stable tests used for benchmarking. These match up with the reference
// implementation's benchmark cases.
var stableTests = []tnetstringTest {
	{map[string]string{}, "0:}"},
	{[]bool{}, "0:]"},
	{"", "0:,"},
	{map[string][]interface{}{"hello": []interface{}{int64(12345678901), "this", true, nil, "\x00\x00\x00\x00"}}, "51:5:hello,39:11:12345678901#4:this,4:true!0:~4:\x00\x00\x00\x00,]}"},
	{12345, "5:12345#"},
	{"this is cool", "12:this is cool,"},
	{nil, "0:~"},
	{true, "4:true!"},
	{false, "5:false!"},
	{"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", "10:\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00,"},
	{[]interface{}{int64(12345), int64(67890), "xxxxx"}, "24:5:12345#5:67890#5:xxxxx,]"},
}

var tnetstringTests = []tnetstringTest {
	{14, "2:14#"},
	{uint(14), "2:14#"},
	{"hello", "5:hello,"},
	{[]int{1, 2, 3}, "12:1:1#1:2#1:3#]"},
	{[...]int{1, 2, 3}, "12:1:1#1:2#1:3#]"},
	{[]string{"ab", "cd", "ef"}, "15:2:ab,2:cd,2:ef,]"},
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

var tests = append(stableTests, tnetstringTests...)

func TestMarshal(t *testing.T) {
	for i, test := range tests {
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
	for i, test := range tests {
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

var jsonData [][]byte
var benchmarkData [][]byte

func init() {
	jsonData = make([][]byte, len(stableTests))
	benchmarkData = make([][]byte, len(stableTests))
	for i, test := range stableTests {
		var err os.Error
		if jsonData[i], err = json.Marshal(test.val); err != nil {
			panic(err.String())
		}
		benchmarkData[i] = []byte(test.data)
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i, test := range benchmarkData {
			ty := reflect.TypeOf(stableTests[i].val)
			if ty == nil {
				continue
			}
			val := reflect.New(ty)
			Unmarshal(test, val.Interface())
		}
	}
}

// JSON benchmark for comparison
func BenchmarkJsonUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i, test := range jsonData {
			ty := reflect.TypeOf(stableTests[i].val)
			if ty == nil {
				continue
			}
			val := reflect.New(ty)
			json.Unmarshal(test, val.Interface())
		}
	}
}
