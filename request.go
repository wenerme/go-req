package req

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

type Request struct {
	Method  string
	BaseURL string
	URL     string
	Query   interface{}
	RawBody []byte
	GetBody func() (io.ReadCloser, error)
	Body    interface{}
	Header  http.Header
	Context context.Context

	Values    Values // Extra options for customized process - non string option use Context
	LastError error
	Options   []interface{}
	Extension Extension
}

type Hook struct {
	Name          string
	Order         int
	OnRequest     func(r *http.Request) error
	OnResponse    func(r *http.Response) error
	HandleRequest func(next http.RoundTripper) http.RoundTripper
	Encode        func(ctx context.Context, body interface{}) ([]byte, error)
	Decode        func(ctx context.Context, body []byte, out interface{}) error
}

type Extension struct {
	Hooks []Hook
}

func (e *Extension) With(h ...Hook) {
	e.Hooks = append(h, e.Hooks...)
	sort.Slice(e.Hooks, func(i, j int) bool {
		// reverse
		return e.Hooks[i].Order > e.Hooks[j].Order
	})
}

func (e Extension) Decode(ctx context.Context, body []byte, out interface{}) error {
	for _, v := range e.Hooks {
		if v.Decode != nil {
			return v.Decode(ctx, body, out)
		}
	}
	return errors.New("no decoder")
}

func (e Extension) Encode(ctx context.Context, body interface{}) ([]byte, error) {
	for _, v := range e.Hooks {
		if v.Encode != nil {
			return v.Encode(ctx, body)
		}
	}
	return nil, errors.New("no encoder")
}

func (e Extension) OnRequest(r *http.Request) error {
	for _, v := range e.Hooks {
		if v.OnRequest != nil {
			if err := v.OnRequest(r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e Extension) RoundTrip(r *http.Request) (*http.Response, error) {
	next := http.DefaultTransport
	for _, v := range e.Hooks {
		if v.HandleRequest != nil {
			next = v.HandleRequest(next)
		}
	}
	resp, err := next.RoundTrip(r)
	return resp, err
}

func (e Extension) OnResponse(r *http.Response) error {
	for _, v := range e.Hooks {
		if v.OnResponse != nil {
			if err := v.OnResponse(r); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewContext(ctx context.Context, r *Request) context.Context {
	return context.WithValue(ctx, (*Request)(nil), r)
}

func FromContext(ctx context.Context) *Request {
	v := ctx.Value((*Request)(nil))
	if v == nil {
		return nil
	}
	return v.(*Request)
}

func (r Request) WithHook(h ...Hook) Request {
	r.Extension.With(h...)
	return r
}

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

	if o.Context != nil {
		r.Context = o.Context
	}

	switch {
	case r.Query == nil:
		r.Query = o.Query
	case o.Query == nil:
		// keep
	default:
		if a, ae := ValuesOf(r.Query); ae == nil {
			if b, be := ValuesOf(o.Query); be == nil {
				r.Query = mergeMapSliceString(a, b)
			} else {
				stdlog.Printf("httmore.RequestInit.Merge: convert query failed %v", be)
			}
		} else {
			stdlog.Printf("httmore.RequestInit.Merge: convert query failed %v", ae)
			r.Query = o.Query
		}
	}

	r.Header = mergeMapSliceString(r.Header, o.Header)
	if r.Values == nil {
		r.Values = Values{}
	}
	r.Values = r.Values.Clone().WithMerge(o.Values)
	r.Options = append(r.Options, o.Options...)
	return r
}

func (r Request) Fetch(out ...interface{}) (*http.Response, error) {
	request, err := r.NewRequest()
	if err != nil {
		return nil, err
	}
	re := FromContext(request.Context())
	response, err := re.Extension.RoundTrip(request)
	if err == nil {
		err = re.Extension.OnResponse(response)
	}
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	all, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	for _, v := range out {
		err = re.Extension.Decode(request.Context(), all, v)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (r Request) NewRequest() (*http.Request, error) {
	if r.LastError != nil {
		return nil, r.LastError
	}
	if err := r.Reconcile(); err != nil {
		return nil, err
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
	for _, o := range r.Options {
		switch v := o.(type) {
		case func(r *Request):
			v(r)
		case func(r *Request) error:
			r.LastError = v(r)
		case Hook:
			r.Extension.With(v)
		default:
			r.LastError = errors.New("invalid option type: " + reflect.TypeOf(o).String())
		}
		if r.LastError != nil {
			return r.LastError
		}
	}

	if r.Method == "" {
		r.Method = http.MethodGet
	}
	{
		u := r.URL
		if strings.HasPrefix(u, "/") && r.BaseURL != "" {
			u = r.BaseURL + u
		}
		if u == "" {
			u = r.BaseURL
		}

		v, err := ValuesOf(r.Query)
		if err != nil {
			return errors.Wrap(err, "build query values")
		}

		parsed, err := url.Parse(u)
		if err != nil {
			return errors.Wrap(err, "invalid url")
		}

		if len(v) > 0 {
			parsed.RawQuery = (url.Values)(mergeMapSliceString(v, parsed.Query())).Encode()
			u = parsed.String()
		}

		r.URL = u
	}

	if r.Context == nil {
		r.Context = context.Background()
	}
	if r.RawBody == nil && r.GetBody == nil && r.Body != nil {
		r.RawBody, r.LastError = r.Extension.Encode(r.Context, r.Body)
	}

	return r.LastError
}
