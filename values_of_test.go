package req

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValuesOf(t *testing.T) {
	for _, test := range []struct {
		v   interface{}
		get func() interface{}
		e   url.Values
	}{
		{v: nil, e: nil},
		{v: (interface{})(nil), e: nil},
		{get: func() interface{} {
			var m map[string][]string
			return &m
		}, e: nil},
		{get: func() interface{} {
			var vv map[string][]string

			m := map[string]interface{}{
				"a": nil,
				"b": &vv,
			}
			return m
		}, e: url.Values{
			"a": []string{""},
			"b": []string{""},
		}},
		{v: map[string]interface{}{
			"a": 1,
			"b": "b",
		}, e: url.Values{
			"a": []string{"1"},
			"b": []string{"b"},
		}},
		{v: struct {
			A string
			B string  `json:"b"`
			C *string `json:"c,omitempty"`
		}{
			"A",
			"B",
			nil,
		}, e: url.Values{
			"A": []string{"A"},
			"b": []string{"B"},
		}},
	} {
		raw := test.v
		if test.get != nil {
			raw = test.get()
		}
		v, err := ValuesOf(raw)
		assert.NoError(t, err)
		assert.Equal(t, test.e, v)
	}
}
