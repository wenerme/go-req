package req

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/pkg/errors"
)

// JSONEncode encode use json.Marshal, add Content-Type
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

// JSONDecode decode use json.Unmarshal
var JSONDecode = Hook{
	Name: "JsonDecode",
	Decode: func(ctx context.Context, body []byte, out interface{}) error {
		return json.Unmarshal(body, out)
	},
}

// FormEncode encode use ValuesOf
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

var MultipartFormEncode = Hook{
	Name: "MultipartFormEncode",
	OnRequest: func(r *http.Request) (err error) {
		if r.Body == nil {
			return errors.New("MultipartFormEncode need file body")
		}
		h := multipart.FileHeader{}
		switch f := r.Body.(type) {
		// case []fs.File:
		case fs.File:
			info, err := f.Stat()
			if err != nil {
				return err
			}
			h.Filename = info.Name()
			h.Size = info.Size()
		default:
			return errors.Errorf("unsupported file type: %T", r.Body)
		}
		r.GetBody = func() (io.ReadCloser, error) {
			b := &bytes.Buffer{}
			// writer := multipart.NewWriter(b)
			// writer.WriteField()
			return io.NopCloser(b), nil
		}
		return
	},
}

// DebugOptions options for DebugHook
type DebugOptions struct {
	Disable   bool                        // Disable turn off debug
	Body      bool                        // Body enable dump http request and response's body
	Out       io.Writer                   // Out debug output, default stderr
	ErrorOnly bool                        // ErrorOnly enable dump error only
	IsError   func(r *http.Response) bool // IsError check if response is error, default is http.StatusOK < 400
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
			if !o.Disable {
				dump, _ := httputil.DumpRequestOut(r, o.Body)
				_, _ = fmt.Fprintln(o.Out, "->", r.Method, r.URL)
				_, _ = fmt.Fprintln(o.Out, string(dump))
			}
			return nil
		},
		OnResponse: func(r *http.Response) error {
			switch {
			case
				o.Disable,
				o.ErrorOnly && o.IsError != nil && !o.IsError(r),
				o.ErrorOnly && o.IsError == nil && r.StatusCode < 400:
			default:
				dump, _ := httputil.DumpResponse(r, o.Body)
				_, _ = fmt.Fprintln(o.Out, "<-", r.Request.Method, r.Request.URL)
				_, _ = fmt.Fprintln(o.Out, string(dump))
			}
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
