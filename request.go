package req

import (
	"bytes"
	"context"
	"io"
	stdlog "log"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// Request is declarative HTTP client instance
type Request struct {
	Method   string
	BaseURL  string
	URL      string
	Query    interface{}
	RawQuery string
	RawBody  []byte
	GetBody  func() (io.ReadCloser, error)
	Body     interface{}
	Header   http.Header
	Context  context.Context

	Values    url.Values // Extra options for customized process - non string option use Context
	LastError error
	// Options support signatures
	//   Request
	//   func(*Request)
	//   func(*Request) error
	//   Hook
	//   nil
	Options   []interface{}
	Extension Extension
}

// WithHook add Hook to Extension
func (r Request) WithHook(h ...Hook) Request {
	r.Extension.With(h...)
	return r
}

// With override Request
func (r Request) With(o Request) Request {
	if o.Method != "" {
		r.Method = o.Method
	}
	if o.BaseURL != "" {
		r.BaseURL = o.BaseURL
	}
	if o.URL != "" {
		r.URL = o.URL
	}

	if o.RawBody != nil {
		r.RawBody = o.RawBody
	}
	if o.Body != nil {
		r.Body = o.Body
	}
	if o.GetBody != nil {
		r.GetBody = o.GetBody
	}

	if o.Context != nil {
		r.Context = o.Context
	}

	switch {
	case o.RawQuery != "":
		r.RawQuery = o.RawQuery
	case r.Query == nil:
		r.Query = o.Query
	case o.Query == nil:
		// keep
	default:
		if a, ae := ValuesOf(r.Query); ae == nil {
			if b, be := ValuesOf(o.Query); be == nil {
				r.Query = mergeMapSliceString(a, b)
			} else {
				stdlog.Printf("req.Request.With: convert query failed %v", be)
			}
		} else {
			stdlog.Printf("req.Request.With: convert query failed %v", ae)
			r.Query = o.Query
		}
	}

	r.Header = mergeMapSliceString(r.Header, o.Header)
	switch {
	case o.Values != nil && r.Values == nil:
		r.Values = o.Values
	case o.Values != nil:
		r.Values = mergeMapSliceString(r.Values, o.Values)
	}
	// run current option first
	r.Options = append(o.Options, r.Options...)
	return r
}

// Do Request
func (r Request) Do() (*http.Response, error) {
	request, err := r.NewRequest()
	if err != nil {
		return nil, err
	}
	re := FromContext(request.Context())
	response, err := re.Extension.RoundTrip(request)
	if err == nil {
		err = re.Extension.OnResponse(response)
	}
	return response, err
}

// FetchBytes return bytes
func (r Request) FetchBytes() ([]byte, *http.Response, error) {
	response, err := r.Do()
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	return body, response, err
}

// FetchString return string
func (r Request) FetchString() (string, *http.Response, error) {
	body, response, err := r.FetchBytes()
	return string(body), response, err
}

// Fetch decode body
func (r Request) Fetch(out ...interface{}) error {
	all, response, err := r.FetchBytes()
	if err != nil {
		return err
	}
	ctx := response.Request.Context()
	re := FromContext(ctx)

	for _, v := range out {
		switch o := v.(type) {
		case **http.Request:
			*o = response.Request
		case **http.Response:
			*o = response
		}
		err = re.Extension.Decode(ctx, all, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewRequest create http.Request
func (r Request) NewRequest() (*http.Request, error) {
	if r.LastError != nil {
		return nil, r.LastError
	}
	if err := r.Reconcile(); err != nil {
		return nil, err
	}
	if r.RawBody == nil && r.GetBody == nil && r.Body != nil {
		r.RawBody, r.LastError = r.Extension.Encode(r.Context, r.Body)
	}
	if r.LastError != nil {
		return nil, r.LastError
	}

	req, err := http.NewRequestWithContext(NewContext(r.Context, &r), r.Method, r.URL, nil)
	if err != nil {
		return nil, err
	}
	if len(r.Header) > 0 {
		req.Header = r.Header
	}
	if len(r.RawBody) > 0 {
		req.Body = io.NopCloser(bytes.NewBuffer(r.RawBody))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBuffer(r.RawBody)), nil
		}
		req.ContentLength = int64(len(r.RawBody))
	} else if r.GetBody != nil {
		b, err := r.GetBody()
		if err != nil {
			return nil, err
		}
		req.Body = b
		req.GetBody = r.GetBody
	}
	return req, r.Extension.OnRequest(req)
}

// Reconcile apply current options, de-sugar request
func (r *Request) Reconcile() error {
	if r.LastError != nil {
		return r.LastError
	}
	// reverse hook order - later hook get higher priority
	l := len(r.Options)
	for i := range r.Options {
		o := r.Options[l-i-1]
		if v, ok := o.(Hook); ok {
			r.Extension.With(v)
		}
	}
	for _, o := range r.Options {
		if o == nil {
			continue
		}
		switch v := o.(type) {
		case Request:
			*r = r.With(v)
		case func(r *Request):
			v(r)
		case func(r *Request) error:
			r.LastError = v(r)
		case Hook:
		default:
			var handled bool
			handled, r.LastError = r.Extension.HandleOption(r, o)
			if !handled {
				r.LastError = errors.New("invalid option type: " + reflect.TypeOf(o).String())
			}
		}
		if r.LastError != nil {
			return r.LastError
		}
	}
	r.Options = nil

	if r.Method == "" {
		r.Method = http.MethodGet
	}
	if r.RawQuery == "" && r.Query != nil {
		v, err := ValuesOf(r.Query)
		if err != nil {
			return errors.Wrap(err, "build query values")
		}
		r.RawQuery = v.Encode()
	}

	{
		u := r.URL
		if strings.HasPrefix(u, "/") && r.BaseURL != "" {
			u = r.BaseURL + u
		}
		if u == "" {
			u = r.BaseURL
		}

		parsed, err := url.Parse(u)
		if err != nil {
			return errors.Wrap(err, "invalid url")
		}

		if r.RawQuery != "" {
			v, err := url.ParseQuery(r.RawQuery)
			if err != nil {
				return errors.Wrap(err, "parse query")
			}
			parsed.RawQuery = (url.Values)(mergeMapSliceString(v, parsed.Query())).Encode()
			u = parsed.String()
		}

		r.URL = u
	}

	if r.Context == nil {
		r.Context = context.Background()
	}

	return r.LastError
}
