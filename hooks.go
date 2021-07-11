package req

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
)

var JSONEncode = Hook{
	Name: "JsonEncode",
	OnRequest: func(r *http.Request) error {
		if r.Header.Get("Content-Type") == "" {
			r.Header.Set("Content-Type", "application/json;charset=UTF-8")
		}
		return nil
	},
	Encode: func(ctx context.Context, body interface{}) ([]byte, error) {
		return json.Marshal(body)
	},
}

var JSONDecode = Hook{
	Name: "JsonDecode",
	Decode: func(ctx context.Context, body []byte, out interface{}) error {
		return json.Unmarshal(body, out)
	},
}

var FormEncode = Hook{
	Name: "FormEncode",
	OnRequest: func(r *http.Request) error {
		if r.Header.Get("Content-Type") == "" {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		return nil
	},
	Encode: func(ctx context.Context, body interface{}) ([]byte, error) {
		v, err := ValuesOf(body)
		if err != nil {
			return nil, err
		}
		return []byte(v.Encode()), nil
	},
}

type DebugOptions struct {
	Body bool
	Out  io.Writer
}

// DebugHook dump http.Request and http.Response
func DebugHook(o *DebugOptions) Hook {
	if o == nil {
		o = &DebugOptions{}
	}
	if o.Out == nil {
		o.Out = os.Stderr
	}
	return Hook{
		Name:  "Debug",
		Order: -100,
		OnRequest: func(r *http.Request) error {
			dump, err := httputil.DumpRequestOut(r, o.Body)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(o.Out, "->", r.Method, r.URL)
			_, _ = fmt.Fprintln(o.Out, string(dump))
			return nil
		},
		OnResponse: func(r *http.Response) error {
			dump, err := httputil.DumpResponse(r, o.Body)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(o.Out, "<-", r.Request.Method, r.Request.URL)
			_, _ = fmt.Fprintln(o.Out, string(dump))
			return nil
		},
	}
}

// UseRoundTripper use customized http.RoundTripper for Request
func UseRoundTripper(rt http.RoundTripper) Hook {
	return Hook{Name: "RoundTripper", Order: -1, HandleRequest: func(next http.RoundTripper) http.RoundTripper {
		return rt
	}}
}
