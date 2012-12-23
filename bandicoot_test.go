package bandicoot

import (
	"io/ioutil"
	"testing"
)

type T1 struct {
	IntAttr  int32
	LongAttr int64
	RealAttr float64
	StrAttr  string
}

func TestSplit(t *testing.T) {
	s := split("a,b,", ',')
	if len(s) != 3 || "a" != s[0] || "b" != s[1] || "" != s[2] {
		t.Errorf("%v\n", s)
	}

	s = split(",", ',')
	if len(s) != 2 || "" != s[0] || "" != s[1] {
		t.Errorf("%v\n", s)
	}

	s = split("", ',')
	if len(s) != 1 || s[0] != "" {
		t.Errorf("%v\n", s)
	}

	s = split("a", ',')
	if len(s) != 1 || s[0] != "a" {
		t.Errorf("%v\n", s)
	}

	s = split("a\\\\,b,:,", ',')
	if len(s) != 4 || "a\\\\" != s[0] || "b" != s[1] || ":" != s[2] || "" != s[3] {
		t.Errorf("%v\n", s)
	}

	s = split("a\\,b,c", ',')
	if len(s) != 2 || "a\\,b" != s[0] || "c" != s[1] {
		t.Errorf("%v\n", s)
	}
}

func equal(v *T1, i int32, l int64, r float64, s string) bool {
	return v != nil && v.IntAttr == i && v.LongAttr == l && v.RealAttr == r && v.StrAttr == s
}

func TestUnmarshal(t *testing.T) {
	var res []T1
	head := "intAttr,longAttr,realAttr,strAttr\n"
	err := unmarshal(head+"\n\n1,1,1.0,hello\n2,2,2.0,world", &res)

	if err != nil {
		t.Errorf("%v\n", err)
	}

	if len(res) != 2 || !equal(&res[0], 1, 1, 1.0, "hello") || !equal(&res[1], 2, 2, 2.0, "world") {
		t.Errorf("comparison failed %v\n", res)
	}

	var res2 []*T1
	err = unmarshal(head+"3,3,3.0,hehe", &res2)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	if len(res2) != 1 || !equal(res2[0], 3, 3, 3.0, "hehe") {
		t.Errorf("comparison failed %v\n", res2)
	}

	err = unmarshal(head, &res)
	if err != nil || len(res) != 0 {
		t.Errorf("%v\n", err)
	}

	err = unmarshal("", nil)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	err = unmarshal(head+"a,3,3.0,hehe", &res2)
	if err == nil {
		t.Errorf("%v\n", err)
	}
}

func TestMarshal(t *testing.T) {
	t1 := T1{0, 1, 2.1, "str"}
	t1h := "intAttr,longAttr,realAttr,strAttr\n"
	t1v := "0,1,2.1,str\n"

	h := marshalHead(t1)
	v := marshalTuple(t1)
	if t1h != h || t1v != v {
		t.Errorf("head: %v, value: %v\n", h, v)
	}

	h = marshalHead(&t1)
	v = marshalTuple(&t1)
	if t1h != h || t1v != v {
		t.Errorf("head: %v, value: %v\n", h, v)
	}

	b, e := marshal([]interface{}{&t1})
	if e != nil {
		t.Errorf("marshal failed: %v", e)
	}

	body, e := ioutil.ReadAll(b)
	if e != nil {
		t.Errorf("read failed: %v", e)
	}

	if t1h+t1v != string(body) {
		t.Errorf("encoded rel: '%v'", string(body))
	}
}

func TestEscapes(t *testing.T) {
	type T1 struct {
		StrAttr string
	}

	h := "StrAttr\n"
	tOrig := T1{"a,b\r\nc,d\n"}
	tEnc := marshalTuple(&tOrig)

	if "a\\,b\r\\\nc\\,d\\\n\n" != tEnc {
		t.Errorf("incorrect value encoding '%v'", tEnc)
	}

	var rel []T1
	err := unmarshal(h+tEnc, &rel)
	if err != nil {
		t.Errorf("failed to unmarshal %s", h+tEnc)
	}

	if len(rel) != 1 || rel[0].StrAttr != tOrig.StrAttr {
		t.Errorf("incorrect value decoding '%v'", rel[0].StrAttr)
	}
}
