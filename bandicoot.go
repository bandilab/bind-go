package bandicoot

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

const (
	Int    = "int"
	Long   = "long"
	Real   = "real"
	String = "string"
)

var url = ""
var headers = make(map[string]string, 0)

func call(method, fn string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url+fn, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return client.Do(req)
}

// Set the bandicoot URL, e.g.:
//  bandicoot.URL("http://localhost:12345")
//  bandicoot.Get("Ping", nil)
func URL(httpURL string) {
	if httpURL[len(httpURL)-1] != '/' {
		httpURL = httpURL + "/"
	}
	url = httpURL
}

// Set an HTTP header. The headers will be passed to bandicoot on each request.
//  bandicoot.SetHeader("X-Auth", "123456789")
func SetHeader(header, value string) {
	headers[header] = value
}

// Call a function using HTTP GET, e.g.:
//  type Book struct {
//    Title string
//    Pages int
//    Price real
//  }
//  var books []Book
//  if err := bandicoot.Get("ListBooks?maxPrice=10.0", &books); err != nil {
//    fmt.Printf("error %v", err)
//  }
//  for _, b := range books {
//    fmt.Printf("%+v\n", b)
//  }
func Get(fn string, out interface{}) error {
	resp, err := call("GET", fn, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("http call failed: %v", resp.Status)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return unmarshal(string(body), out)
}

// Call a function using HTTP POST, e.g.:
//  books := []Book{Book{Title: "Robinson Crusoe", Pages: 312, Price: 11.21}}
//  bandicoot.Post("AddBooks", books, nil)
func Post(fn string, in []interface{}, out interface{}) error {
	reqBody, err := marshal(in)
	if err != nil {
		return err
	}

	resp, err := call("POST", fn, reqBody)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("http call failed: %v", resp.Status)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return unmarshal(string(respBody), out)
}

func marshal(args []interface{}) (io.Reader, error) {
	var buf *bytes.Buffer = bytes.NewBufferString("")
	if len(args) > 0 {
		h := marshalHead(args[0])
		if len(h) > 0 {
			buf.WriteString(h)
			for i := 0; i < len(args); i++ {
				t := marshalTuple(args[i])
				if len(t) == 0 {
					return nil, fmt.Errorf("cannot marshal the value %v", args[i])
				}

				buf.WriteString(t)
			}
		} else {
			return nil, fmt.Errorf("cannot marshal the header: %v", args[0])
		}
	}

	return buf, nil
}

func attrUpper(s string) string {
	if len(s) == 0 {
		return s
	}

	b := []rune(s)
	if unicode.IsLower(b[0]) {
		b[0] = unicode.ToUpper(b[0])
	}

	return string(b)
}

func attrLower(s string) string {
	if len(s) == 0 {
		return s
	}

	b := []rune(s)
	if unicode.IsUpper(b[0]) {
		b[0] = unicode.ToLower(b[0])
	}

	return string(b)
}

func stripComa(s string) string {
	if s[len(s)-1] == ',' {
		s = s[0 : len(s)-1]
	}

	return s
}

func marshalHead(v interface{}) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return ""
	}

	res := ""
	last := t.NumField() - 1
	for i := 0; i <= last; i++ {
		a := t.Field(i)
		if a.Type.Kind() == reflect.Chan {
			continue
		}

		res += attrLower(a.Name) + ","
	}

	return stripComa(res) + "\n"
}

func marshalTuple(t interface{}) string {
	v := reflect.ValueOf(t)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	res := ""
	last := v.NumField() - 1
	for i := 0; i <= last; i++ {
		if v.Field(i).Kind() == reflect.Chan || !v.Field(i).CanInterface() {
			continue
		}

		value := fmt.Sprintf("%v", v.Field(i).Interface())
		value = strings.Replace(value, ",", "\\,", -1)
		value = strings.Replace(value, "\n", "\\\n", -1)

		res += value + ","
	}

	return stripComa(res) + "\n"
}

func unmarshalTuple(attrs []string, line string, out reflect.Value) error {
	if out.Kind() != reflect.Ptr || out.Elem().Kind() == reflect.Ptr {
		return fmt.Errorf("expected a pointer type, got '%v'", out.Type())
	}

	fields := 0
	out = reflect.Indirect(out)

	for i, strVal := range split(line, ',') {
		if i < len(attrs) {
			strVal = strings.Replace(strVal, "\\,", ",", -1)
			strVal = strings.Replace(strVal, "\\\n", "\n", -1)

			attr := out.FieldByName(attrs[i])
			if !attr.IsValid() || !attr.CanSet() {
				return fmt.Errorf("cannot set attribute '%s'", attrs[i])
			}

			switch attr.Type().Kind() {
			case reflect.Int, reflect.Uint, reflect.Int32, reflect.Uint32:
				v, err := strconv.ParseInt(strVal, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid integer value %s:'%s'", attrs[i], strVal)
				}

				attr.SetInt(v)
			case reflect.Int64, reflect.Uint64:
				v, err := strconv.ParseInt(strVal, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid long value %s:'%s'", attrs[i], strVal)
				}

				attr.SetInt(v)
			case reflect.Float32, reflect.Float64:
				v, err := strconv.ParseFloat(strVal, 64)
				if err != nil {
					return fmt.Errorf("invalid real value %s:'%s'", attrs[i], strVal)
				}

				attr.SetFloat(v)
			case reflect.String:
				attr.SetString(strVal)
			default:
				return fmt.Errorf("invalid attribute type %s:%v", attrs[i], attr.Type())
			}
			fields++
		} else {
			return fmt.Errorf("tuple contains more attributes than expected '%v'", line)
		}
	}

	if fields != len(attrs) {
		return fmt.Errorf("tuple contains less attributes than expected '%v'", line)
	}

	return nil
}

func unmarshal(rel string, v interface{}) error {
	out := reflect.ValueOf(v)
	if strings.TrimSpace(rel) == "" && !out.IsValid() {
		return nil
	}

	if out.Kind() != reflect.Ptr {
		return fmt.Errorf("output should be a pointer (got '%v')", reflect.TypeOf(v))
	}

	lines := split(rel, '\n')
	attrs := make([]string, 0)
	for _, a := range split(lines[0], ',') {
		attrs = append(attrs, attrUpper(a))
	}

	out = reflect.Indirect(out)
	out.Set(reflect.MakeSlice(out.Type(), 0, len(lines)-1))

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}

		t := out.Type().Elem()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		tuple := reflect.New(t)
		if err := unmarshalTuple(attrs, lines[i], tuple); err != nil {
			return err
		}

		pos := out.Len()
		out.SetLen(pos + 1)

		elem := out.Index(pos)
		if elem.Kind() == reflect.Ptr {
			elem.Set(tuple)
		} else {
			elem.Set(reflect.Indirect(tuple))
		}
	}

	return nil
}

func split(str string, sep byte) []string {
	const esc = byte('\\')
	prev := byte('\000')
	res := make([]string, 0)
	elem := make([]byte, 0)

	for i := 0; i < len(str); i++ {
		if str[i] == sep && prev != esc {
			res = append(res, string(elem))
			elem = make([]byte, 0)
		} else {
			elem = append(elem, str[i])
		}

		if prev == esc {
			prev = byte('\000')
		} else {
			prev = str[i]
		}
	}

	if len(str) == 0 || len(elem) != 0 || (str[len(str)-1] == sep && prev != esc) {
		res = append(res, string(elem))
	}

	return res
}
