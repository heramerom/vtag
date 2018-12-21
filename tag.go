package vtag

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var cache sync.Map

func sliceContains(ss []string, s string) (b bool) {
	for _, v := range ss {
		if v == s {
			b = true
			return
		}
	}
	return
}

func sliceHasIntersection(s1, s2 []string) bool {
	for _, t := range s1 {
		if sliceContains(s2, t) {
			return true
		}
	}
	return false
}

type EncoderFunc func(field reflect.StructTag, pre string, name string) (key string)

var UpperEncodeFunc = EncoderFunc(func(field reflect.StructTag, pre string, name string) (key string) {
	if pre != "" {
		return pre + "." + strings.ToUpper(name)
	}
	return strings.ToUpper(name)
})

var LowerEncodeFunc = EncoderFunc(func(field reflect.StructTag, pre string, name string) (key string) {
	if pre != "" {
		return pre + "." + strings.ToLower(name)
	}
	return strings.ToLower(name)
})

var CamelCaseEncodeFunc = EncoderFunc(func(field reflect.StructTag, pre string, name string) (key string) {
	if name[0] >= 'A' && name[0] <= 'Z' {
		name = string([]byte{name[0] - 'A' + 'a'}) + name[1:]
	}
	if pre != "" {
		return pre + "." + name
	}
	return name
})

var UnderScoreCaseEncodeFunc = EncoderFunc(func(field reflect.StructTag, pre string, name string) (key string) {
	if pre != "" {
		return pre + "." + Snake2UnderScoreCase(name)
	}
	return Snake2UnderScoreCase(name)
})

func Snake2UnderScoreCase(snake string) string {
	var last, lastWrite byte
	buf := bytes.NewBuffer(nil)
	for idx, v := range []byte(snake) {
		if idx == 0 {
			last = v
			continue
		}
		if isLowerByte(v) {
			if isUpperByte(last) && idx != 1 && lastWrite != '_' {
				buf.WriteByte('_')
			}
		}
		buf.WriteByte(lowerByte(last))
		lastWrite = last
		if isLowerByte(last) && isUpperByte(v) {
			buf.WriteByte('_')
			lastWrite = '_'
		}
		last = v
	}
	buf.WriteByte(lowerByte(last))
	return buf.String()
}

func lowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

func isLowerByte(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func isUpperByte(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isDigitalByte(b byte) bool {
	return b >= '0' && b <= '9'
}

func SliceWithTag(obj interface{}, pre string, tags ...string) (names []string, err error) {
	return getDefaultEncoder().SliceWithTag(obj, pre, tags...)
}

func Slice2Map(s []string, value interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(s))
	for _, v := range s {
		m[v] = value
	}
	return m
}

type Encoder interface {
	SliceWithTag(obj interface{}, pre string, tags ...string) (names []string, err error)
}

var defaultEncoder Encoder

type encode struct {
	tag     string
	encoder EncoderFunc
	cache   *sync.Map
}

func NewEncoder(tag string, enc EncoderFunc, cache bool) Encoder {
	if tag == "" {
		tag = "vtag"
	}
	e := &encode{
		tag:     tag,
		encoder: enc,
	}
	if cache {
		e.cache = &sync.Map{}
	}
	return e
}

var once sync.Once

func InitEncoder(enc EncoderFunc) {
	once.Do(func() {
		defaultEncoder = NewEncoder("", enc, true)
	})
}

func getDefaultEncoder() Encoder {
	once.Do(func() {
		defaultEncoder = NewEncoder("", nil, true)
	})
	return defaultEncoder
}

func (e *encode) SliceWithTag(obj interface{}, pre string, tags ...string) (names []string, err error) {
	var typ reflect.Type
	if v, ok := obj.(reflect.Type); ok {
		typ = v
	} else {
		typ = reflect.TypeOf(obj)
	}
	var cacheKey string
	if e.cache != nil {
		cacheKey = typ.PkgPath() + typ.Name() + strings.Join(tags, "")
		if cacheKey != "" {
			if v, ok := e.cache.Load(cacheKey); ok {
				names = v.([]string)
				return
			}
		}
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		err = fmt.Errorf("type: %s, not support", typ.Kind())
		return
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// anonymous
		if field.Anonymous {
			m, _ := e.SliceWithTag(field.Type, pre, tags...)
			names = append(names, m...)
			continue
		}
		name := field.Name
		if name[0] < 'A' || name[0] > 'Z' {
			continue
		}
		ts := strings.Split(field.Tag.Get(e.tag), ",")
		if ts[0] == "-" {
			continue
		}
		if !sliceHasIntersection(tags, ts) {
			continue
		}
		// name
		if ts[0] != "" {
			name = ts[0]
			if pre != "" {
				name = pre + "." + ts[0]
			}
		} else if e.encoder != nil {
			name = e.encoder(field.Tag, pre, field.Name)
			if name == "" {
				continue
			}
		} else if pre != "" {
			name = pre + "." + name
		}
		// ptr
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		// struct
		if t.Kind() == reflect.Struct {
			m, _ := SliceWithTag(t, name, tags...)
			if m != nil {
				names = append(names, m...)
			}
			continue
		}
		names = append(names, name)
	}
	if e.cache != nil {
		if cacheKey != "" {
			e.cache.Store(cacheKey, names)
		}
	}
	return
}
