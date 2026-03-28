package req

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type ReqP struct{}

type Par struct {
	A int `json:"a"`
}

func TestValuesOf(t *testing.T) {
	var nilP *ReqP
	now := time.Now()
	for _, test := range []struct {
		v   any
		get func() any
		e   url.Values
	}{
		{v: nil, e: nil},
		{v: &Par{A: 1000000000018}, e: url.Values{"a": []string{"1000000000018"}}},
		{v: map[string]any{"v": 1000000000018}, e: url.Values{"v": []string{"1000000000018"}}},
		{v: nilP, e: nil},
		{v: url.Values{}, e: url.Values{}},
		{v: (any)(nil), e: nil},
		{get: func() any {
			var m map[string][]string
			return &m
		}, e: nil},
		{get: func() any {
			var vv map[string][]string

			m := map[string]any{
				"a": nil,
				"b": &vv,
			}
			return m
		}, e: url.Values{
			"a": []string{""},
			"b": []string{""},
		}},
		{v: map[string]any{
			"t": now,
		}, e: url.Values{
			"t": []string{now.Format(time.RFC3339)},
		}},
		{v: map[string]any{
			"v": []any{nil, "v"},
		}, e: url.Values{
			"v": []string{"", "v"},
		}},
		{v: map[string]any{
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

	_, err := ValuesOf(1)
	assert.Error(t, err)
	_, err = ValuesOf(1.1)
	assert.Error(t, err)
	_, err = ValuesOf("")
	assert.Error(t, err)
}
