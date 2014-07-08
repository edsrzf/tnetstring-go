package tnetstring

import (
	"encoding/json"
	"reflect"
	"testing"
)

type tnetstringTest struct {
	val  interface{}
	data string
}

// Stable tests used for benchmarking. These match up with the reference
// implementation's benchmark cases.
var stableTests = []tnetstringTest{
	{map[string]string{}, "0:}"},
	{[]bool(nil), "0:]"},
	{map[string][]interface{}{"hello": []interface{}{int64(12345678901), "this", true, nil, "\x00\x00\x00\x00"}}, "51:5:hello,39:11:12345678901#4:this,4:true!0:~4:\x00\x00\x00\x00,]}"},
	{12345, "5:12345#"},
	{"this is cool", "12:this is cool,"},
	{"", "0:,"},
	{nil, "0:~"},
	{true, "4:true!"},
	{false, "5:false!"},
	{"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", "10:\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00,"},
	{[]interface{}{int64(12345), int64(67890), "xxxxx"}, "24:5:12345#5:67890#5:xxxxx,]"},
	{[]float64{0.1, 0.2, 0.3}, "18:3:0.1^3:0.2^3:0.3^]"},
	{
		[][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][][]string{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{"hello-there"}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}},
		"243:238:233:228:223:218:213:208:203:198:193:188:183:178:173:168:163:158:153:148:143:138:133:128:123:118:113:108:103:99:95:91:87:83:79:75:71:67:63:59:55:51:47:43:39:35:31:27:23:19:15:11:hello-there,]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]",
	},
}

var tnetstringTests = []tnetstringTest{
	{14, "2:14#"},
	{-1, "2:-1#"},
	{1000000000, "10:1000000000#"},
	{uint(14), "2:14#"},
	{uint(1000000000), "10:1000000000#"},
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
		out, err := Marshal(test.val)
		if err != nil {
			t.Errorf("#%d Marshal error: %s", i, err)
			continue
		}
		if out != test.data {
			t.Errorf("#%d want\n%q\ngot\n%q", i, test.data, out)
		}
	}
}

type mapStruct struct {
	A map[string]string
	B string
}

var mapTests = []tnetstringTest{
	{map[string]int{"a": 1, "b": 2}, "16:1:a,1:1#1:b,1:2#}"},
	{map[string]mapStruct{
		"k1": mapStruct{A: map[string]string{"a": "b", "c": "d"}, B: "str1"},
		"k2": mapStruct{A: map[string]string{"e": "f", "g": "h"}, B: "str2"},
	}, "88:2:k1,35:1:A,16:1:a,1:b,1:c,1:d,}1:B,4:str1,}2:k2,35:1:A,16:1:e,1:f,1:g,1:h,}1:B,4:str2,}}"},
	{map[string]map[string]map[string]string{
		"k1": map[string]map[string]string{"k3": map[string]string{"a": "b", "c": "d"}, "k4": map[string]string{"i": "j"}},
		"k2": map[string]map[string]string{"k5": map[string]string{"e": "f", "g": "h"}, "k6": map[string]string{"k": "l"}},
	}, "100:2:k1,41:2:k3,16:1:a,1:b,1:c,1:d,}2:k4,8:1:i,1:j,}}2:k2,41:2:k5,16:1:e,1:f,1:g,1:h,}2:k6,8:1:k,1:l,}}}"},
}

func TestUnmarshal(t *testing.T) {

	tests = append(tests, mapTests...)

	for i, test := range tests {
		ty := reflect.TypeOf(test.val)
		if ty == nil {
			continue
		}
		val := reflect.New(ty)
		rest, err := Unmarshal(test.data, val.Interface())
		if err != nil {
			t.Errorf("#%d Unmarshal error: %s", i, err)
		}
		if rest != "" {
			t.Errorf("#%d Unmarshal returned non-empty left-over: %v", i, rest)
		}
		if !reflect.DeepEqual(test.val, val.Elem().Interface()) {
			t.Errorf("#%d want\n%v\ngot\n%v", i, test.val, val.Elem().Interface())
		}
	}
}

var jsonData [][]byte
var benchmarkData []string

func init() {
	jsonData = make([][]byte, len(stableTests))
	benchmarkData = make([]string, len(stableTests))
	for i, test := range stableTests {
		var err error
		if jsonData[i], err = json.Marshal(test.val); err != nil {
			panic(err.Error())
		}
		benchmarkData[i] = test.data
	}
}

func BenchmarkMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, test := range stableTests {
			Marshal(test.val)
		}
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

// JSON benchmarks for comparison

func BenchmarkJsonMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, test := range stableTests {
			json.Marshal(test.val)
		}
	}
}

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
