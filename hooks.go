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
	Encode: func(ctx context.Context, body any) ([]byte, error) {
		return json.Marshal(body)
	},
}

// JSONDecode decode use json.Unmarshal
var JSONDecode = Hook{
	Name: "JsonDecode",
	Decode: func(ctx context.Context, body []byte, out any) error {
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
	Encode: func(ctx context.Context, body any) ([]byte, error) {
		v, err := ValuesOf(body)
		if err != nil {
			return nil, err
		}
		return []byte(v.Encode()), nil
	},
}

// MultipartFile represents a file to be uploaded via multipart form
type MultipartFile struct {
	// FieldName is the form field name (default: "file")
	FieldName string
	// Filename is the filename to use in the multipart header
	Filename string
	// Reader provides the file content
	Reader io.Reader
	// Fields are extra form fields to include
	Fields map[string]string
}

// MultipartFormEncode encodes a MultipartFile or fs.File as multipart/form-data.
// Body must be *MultipartFile or fs.File.
var MultipartFormEncode = Hook{
	Name: "MultipartFormEncode",
	Encode: func(ctx context.Context, body any) ([]byte, error) {
		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)

		var fieldName, filename string
		var reader io.Reader
		var fields map[string]string

		switch f := body.(type) {
		case *MultipartFile:
			fieldName = f.FieldName
			filename = f.Filename
			reader = f.Reader
			fields = f.Fields
		case fs.File:
			info, err := f.Stat()
			if err != nil {
				return nil, fmt.Errorf("stat file: %w", err)
			}
			filename = info.Name()
			reader = f
		default:
			return nil, fmt.Errorf("MultipartFormEncode: unsupported body type %T, use *MultipartFile or fs.File", body)
		}

		if fieldName == "" {
			fieldName = "file"
		}
		if filename == "" {
			filename = "upload"
		}

		// Write extra fields first
		for k, v := range fields {
			if err := writer.WriteField(k, v); err != nil {
				return nil, fmt.Errorf("write field %s: %w", k, err)
			}
		}

		// Write file part
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, reader); err != nil {
			return nil, fmt.Errorf("copy file content: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close multipart writer: %w", err)
		}

		// Store content type in context for OnRequest to pick up
		return buf.Bytes(), nil
	},
	OnRequest: func(r *http.Request) error {
		// Detect multipart content and set proper boundary
		if r.Body != nil && r.Header.Get("Content-Type") == "" {
			// Re-encode to get boundary — need to parse from body
			// Simpler: just check if body looks like multipart
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			if bytes.HasPrefix(body, []byte("--")) {
				// Extract boundary from first line
				idx := bytes.IndexByte(body, '\r')
				if idx < 0 {
					idx = bytes.IndexByte(body, '\n')
				}
				if idx > 2 {
					boundary := string(body[2:idx])
					r.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
		}
		return nil
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
