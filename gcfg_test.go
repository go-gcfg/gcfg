package gcfg

import (
	"reflect"
	"testing"
)

const (
	// 64 spaces
	sp64 = "                                                                "
	// 512 spaces
	sp512 = sp64 + sp64 + sp64 + sp64 + sp64 + sp64 + sp64 + sp64
	// 4096 spaces
	sp4096 = sp512 + sp512 + sp512 + sp512 + sp512 + sp512 + sp512 + sp512
)

type sect01 struct{ Name string }
type conf01 struct{ Section sect01 }

type sect02 struct{ Bool bool }
type conf02 struct{ Section sect02 }

var parsetests = []struct {
	gcfg string
	exp  interface{}
	ok   bool
}{
	// from ExampleParseString
	{"[section]\nname=value", &conf01{sect01{"value"}}, true},
	// non-string value
	{"[section]\nbool=true", &conf02{sect02{true}}, true},
	// error: line too long 
	{"[section]\nname=value\n" + sp4096, &conf01{}, false},
	// error: no section
	{"name=value", &conf01{}, false},
}

func TestParse(t *testing.T) {
	for i, tt := range parsetests {
		// get the type of the expected result 
		restyp := reflect.TypeOf(tt.exp).Elem()
		// create a new instance to hold the actual result
		res := reflect.New(restyp).Interface()
		err := ParseString(res, tt.gcfg)
		if tt.ok {
			if err != nil {
				t.Errorf("#%d fail: got error %#v, wanted ok", i, err)
			} else if !reflect.DeepEqual(res, tt.exp) {
				t.Errorf("#%d fail: got %#v, wanted %#v", i, res, tt.exp)
			}
			t.Logf("#%d pass: ok, %#v", i, res)
		} else { // !tt.ok
			if err == nil {
				t.Errorf("#%d fail: got %#v, wanted error", i, res)
			}
			t.Logf("#%d pass: !ok, %#v", i, err)
		}
	}
}

func TestParseFile(t *testing.T) {
	res := &struct{ Section struct{ Name string } }{}
	err := ParseFile(res, "gcfg_test.gcfg")
	if err != nil {
		t.Fatal(err)
	}
	if "value" != res.Section.Name {
		t.Errorf("got %q, wanted %q", res.Section.Name, "value")
	}
}
