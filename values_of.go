package req

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

// ValuesOf convert anything to url.Values
func ValuesOf(v interface{}) (url.Values, error) {
	if v == nil {
		return nil, nil
	}
	switch tv := v.(type) {
	case url.Values:
		return tv, nil
	case map[string][]string:
		return tv, nil
	case map[string]string:
		return mapStringToSliceString(tv), nil
	}
	rv := reflect.ValueOf(v)
	changed := false
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
		changed = true
	}
	if !rv.IsValid() {
		return nil, nil
	}
	if changed {
		return ValuesOf(rv.Interface())
	}
	if rv.Kind() == reflect.Struct {
		// json-ize - use json tag
		v, err := json.Marshal(v)
		if err == nil {
			var m map[string]interface{}
			err = json.Unmarshal(v, &m)
			if err == nil {
				return ValuesOf(m)
			}
		}
		return nil, err
	}
	if rv.Kind() == reflect.Map {
		m := make(url.Values)
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			sv := iter.Value()
			if sv.Kind() == reflect.Interface {
				sv = reflect.ValueOf(sv.Interface())
			}
			switch sv.Kind() {
			case
				reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer,
				reflect.Interface, reflect.Slice:
				if sv.IsNil() {
					continue
				}
			}
			switch {
			case sv.Kind() == reflect.Slice || sv.Kind() == reflect.Array:
				// no empty check
				for i := 0; i < sv.Len(); i++ {
					m.Add(fmt.Sprint(k.Interface()), valueString(sv.Index(i)))
				}
			default:
				m.Set(fmt.Sprint(k.Interface()), valueString(sv))
			}
		}
		return m, nil
	}
	return nil, fmt.Errorf("ValuesOf: unsupported type %T", v)
}

func valueString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.IsZero() {
		return ""
	}

	if v.Type() == timeType {
		t := v.Interface().(time.Time)
		return t.Format(time.RFC3339)
	}
	if v.CanFloat() {
		// prevent 1.000000000018e+12
		val := v.Float()
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10) //nolint:gomnd
		}
		return fmt.Sprintf("%f", val)
	}
	return fmt.Sprint(v.Interface())
}

func mapStringToSliceString(a map[string]string) map[string][]string {
	if len(a) == 0 {
		return nil
	}
	m := make(map[string][]string)
	for k, v := range a {
		m[k] = []string{v}
	}
	return m
}
