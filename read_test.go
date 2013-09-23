package gcfg

import (
	"fmt"
	"os"
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

type cBasic struct {
	Section           cBasicS1
	Hyphen_In_Section cBasicS2
}
type cBasicS1 struct {
	Name string
	Int  int
}
type cBasicS2 struct {
	Hyphen_In_Name string
}

type nonMulti []string

type unmarshalable string

func (u *unmarshalable) UnmarshalText(text []byte) error {
	s := string(text)
	if s == "error" {
		return fmt.Errorf("%s", s)
	}
	*u = unmarshalable(s)
	return nil
}

var _ textUnmarshaler = new(unmarshalable)

type cMulti struct {
	M1 cMultiS1
	M2 cMultiS2
	M3 cMultiS3
}
type cMultiS1 struct{ Multi []string }
type cMultiS2 struct{ NonMulti nonMulti }
type cMultiS3 struct{ MultiInt []int }

type cSubs struct{ Sub map[string]*cSubsS1 }
type cSubsS1 struct{ Name string }

type cBool struct{ Section cBoolS1 }
type cBoolS1 struct{ Bool bool }

type cTxUnm struct{ Section cTxUnmS1 }
type cTxUnmS1 struct{ Name unmarshalable }

type cNum struct {
	N1 cNumS1
	N2 cNumS2
	N3 cNumS3
}
type cNumS1 struct{ Int int }
type cNumS2 struct{ MultiInt []int }
type cNumS3 struct{ FileMode os.FileMode }

type readtest struct {
	gcfg string
	exp  interface{}
	ok   bool
}

var readtests = []struct {
	group string
	tests []readtest
}{{"scanning", []readtest{
	{"[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	// hyphen in name
	{"[hyphen-in-section]\nhyphen-in-name=value", &cBasic{Hyphen_In_Section: cBasicS2{Hyphen_In_Name: "value"}}, true},
	// quoted string value
	{"[section]\nname=\"\"", &cBasic{Section: cBasicS1{Name: ""}}, true},
	{"[section]\nname=\" \"", &cBasic{Section: cBasicS1{Name: " "}}, true},
	{"[section]\nname=\"value\"", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=\" value \"", &cBasic{Section: cBasicS1{Name: " value "}}, true},
	{"\n[section]\nname=\"va ; lue\"", &cBasic{Section: cBasicS1{Name: "va ; lue"}}, true},
	{"[section]\nname=\"val\" \"ue\"", &cBasic{Section: cBasicS1{Name: "val ue"}}, true},
	{"[section]\nname=\"value", &cBasic{}, false},
	// escape sequences
	{"[section]\nname=\"va\\\\lue\"", &cBasic{Section: cBasicS1{Name: "va\\lue"}}, true},
	{"[section]\nname=\"va\\\"lue\"", &cBasic{Section: cBasicS1{Name: "va\"lue"}}, true},
	{"[section]\nname=\"va\\nlue\"", &cBasic{Section: cBasicS1{Name: "va\nlue"}}, true},
	{"[section]\nname=\"va\\tlue\"", &cBasic{Section: cBasicS1{Name: "va\tlue"}}, true},
	{"\n[section]\nname=\\", &cBasic{}, false},
	{"\n[section]\nname=\\a", &cBasic{}, false},
	{"\n[section]\nname=\"val\\a\"", &cBasic{}, false},
	{"\n[section]\nname=val\\", &cBasic{}, false},
	{"\n[sub \"A\\\n\"]\nname=value", &cSubs{}, false},
	{"\n[sub \"A\\\t\"]\nname=value", &cSubs{}, false},
	// broken line
	{"[section]\nname=value \\\n value", &cBasic{Section: cBasicS1{Name: "value  value"}}, true},
	{"[section]\nname=\"value \\\n value\"", &cBasic{}, false},
}}, {"scanning:whitespace", []readtest{
	{" \n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{" [section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\t[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[ section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section ]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\n name=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname =value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname= value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=value ", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\r\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\r\nname=value\r\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{";cmnt\r\n[section]\r\nname=value\r\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	// long lines
	{sp4096 + "[section]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[" + sp4096 + "section]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section" + sp4096 + "]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]" + sp4096 + "\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\n" + sp4096 + "name=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname" + sp4096 + "=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=" + sp4096 + "value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=value\n" + sp4096, &cBasic{Section: cBasicS1{Name: "value"}}, true},
}}, {"scanning:comments", []readtest{
	{"; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"# cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{" ; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\t; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]; cmnt\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section] ; cmnt\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value ; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=\"value\" ; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value ; \"cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=\"va ; lue\" ; cmnt", &cBasic{Section: cBasicS1{Name: "va ; lue"}}, true},
	{"\n[section]\nname=; cmnt", &cBasic{Section: cBasicS1{Name: ""}}, true},
}}, {"scanning:subsections", []readtest{
	{"\n[sub \"A\"]\nname=value", &cSubs{map[string]*cSubsS1{"A": &cSubsS1{"value"}}}, true},
	{"\n[sub \"b\"]\nname=value", &cSubs{map[string]*cSubsS1{"b": &cSubsS1{"value"}}}, true},
	{"\n[sub \"A\\\\\"]\nname=value", &cSubs{map[string]*cSubsS1{"A\\": &cSubsS1{"value"}}}, true},
	{"\n[sub \"A\\\"\"]\nname=value", &cSubs{map[string]*cSubsS1{"A\"": &cSubsS1{"value"}}}, true},
}}, {"syntax", []readtest{
	// invalid line
	{"\n[section]\n=", &cBasic{}, false},
	// no section
	{"name=value", &cBasic{}, false},
	// empty section
	{"\n[]\nname=value", &cBasic{}, false},
	// empty subsection
	{"\n[sub \"\"]\nname=value", &cSubs{}, false},
}}, {"setting", []readtest{
	{"[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	// section name not matched
	{"\n[nonexistent]\nname=value", &cBasic{}, false},
	// subsection name not matched
	{"\n[section \"nonexistent\"]\nname=value", &cBasic{}, false},
	// variable name not matched
	{"\n[section]\nnonexistent=value", &cBasic{}, false},
	// hyphen in name
	{"[hyphen-in-section]\nhyphen-in-name=value", &cBasic{Hyphen_In_Section: cBasicS2{Hyphen_In_Name: "value"}}, true},
}}, {"multivalue", []readtest{
	// unnamed slice type: treat as multi-value
	{"\n[m1]", &cMulti{M1: cMultiS1{}}, true},
	{"\n[m1]\nmulti=value", &cMulti{M1: cMultiS1{[]string{"value"}}}, true},
	{"\n[m1]\nmulti=value1\nmulti=value2", &cMulti{M1: cMultiS1{[]string{"value1", "value2"}}}, true},
	// named slice type: do not treat as multi-value
	{"\n[m2]", &cMulti{}, true},
	{"\n[m2]\nmulti=value", &cMulti{}, false},
	{"\n[m2]\nmulti=value1\nmulti=value2", &cMulti{}, false},
}}, {"type:string", []readtest{
	{"[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=", &cBasic{Section: cBasicS1{Name: ""}}, true},
}}, {"type:bool", []readtest{
	// explicit values
	{"[section]\nbool=true", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=yes", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=on", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=1", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=false", &cBool{cBoolS1{false}}, true},
	{"[section]\nbool=no", &cBool{cBoolS1{false}}, true},
	{"[section]\nbool=off", &cBool{cBoolS1{false}}, true},
	{"[section]\nbool=0", &cBool{cBoolS1{false}}, true},
	// implicit value (true)
	{"[section]\nbool", &cBool{cBoolS1{true}}, true},
	// bool parse errors
	{"[section]\nbool=maybe", &cBool{}, false},
	{"[section]\nbool=t", &cBool{}, false},
	{"[section]\nbool=truer", &cBool{}, false},
	{"[section]\nbool=2", &cBool{}, false},
	{"[section]\nbool=-1", &cBool{}, false},
}}, {"type:numeric", []readtest{
	{"[section]\nint=0", &cBasic{Section: cBasicS1{Int: 0}}, true},
	{"[section]\nint=1", &cBasic{Section: cBasicS1{Int: 1}}, true},
	{"[section]\nint=-1", &cBasic{Section: cBasicS1{Int: -1}}, true},
	{"[section]\nint=0.2", &cBasic{}, false},
	{"[section]\nint=1e3", &cBasic{}, false},
	// primitive [u]int(|8|16|32|64) is parsed as decimal (not octal)
	{"[n1]\nint=010", &cNum{N1: cNumS1{Int: 10}}, true},
	{"[n2]\nmultiint=010", &cNum{N2: cNumS2{MultiInt: []int{10}}}, true},
	// octal allowed for named type
	{"[n3]\nfilemode=0777", &cNum{N3: cNumS3{FileMode: 0777}}, true},
}}, {"type:textUnmarshaler", []readtest{
	{"[section]\nname=value", &cTxUnm{Section: cTxUnmS1{Name: "value"}}, true},
	{"[section]\nname=error", &cTxUnm{}, false},
}},
}

func TestReadStringInto(t *testing.T) {
	for _, tg := range readtests {
		for i, tt := range tg.tests {
			id := fmt.Sprintf("%s:%d", tg.group, i)
			testRead(t, id, tt)
		}
	}
}

func testRead(t *testing.T, id string, tt readtest) {
	// get the type of the expected result
	restyp := reflect.TypeOf(tt.exp).Elem()
	// create a new instance to hold the actual result
	res := reflect.New(restyp).Interface()
	err := ReadStringInto(res, tt.gcfg)
	if tt.ok {
		if err != nil {
			t.Errorf("%s fail: got error %v, wanted ok", id, err)
			return
		} else if !reflect.DeepEqual(res, tt.exp) {
			t.Errorf("%s fail: got value %#v, wanted value %#v", id, res, tt.exp)
			return
		}
		if !testing.Short() {
			t.Logf("%s pass: got value %#v", id, res)
		}
	} else { // !tt.ok
		if err == nil {
			t.Errorf("%s fail: got value %#v, wanted error", id, res)
			return
		}
		if !testing.Short() {
			t.Logf("%s pass: got error %v", id, err)
		}
	}
}

func TestReadFileInto(t *testing.T) {
	res := &struct{ Section struct{ Name string } }{}
	err := ReadFileInto(res, "testdata/gcfg_test.gcfg")
	if err != nil {
		t.Errorf(err.Error())
	}
	if "value" != res.Section.Name {
		t.Errorf("got %q, wanted %q", res.Section.Name, "value")
	}
}
