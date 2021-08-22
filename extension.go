package req

import (
	"context"
	"net/http"
	"sort"

	"github.com/pkg/errors"
)

// Hook phases for Extension
type Hook struct {
	Name          string
	Order         int
	OnRequest     func(r *http.Request) error
	OnResponse    func(r *http.Response) error
	HandleRequest func(next http.RoundTripper) http.RoundTripper
	HandleOption  func(r *Request, o interface{}) (bool, error)
	Encode        func(ctx context.Context, body interface{}) ([]byte, error)
	Decode        func(ctx context.Context, body []byte, out interface{}) error
}

// Extension of Request
type Extension struct {
	Hooks []Hook
}

// With more hooks
func (e *Extension) With(h ...Hook) {
	e.Hooks = append(h, e.Hooks...)
	sort.Slice(e.Hooks, func(i, j int) bool {
		// reverse
		return e.Hooks[i].Order > e.Hooks[j].Order
	})
}

// Decode body
func (e Extension) Decode(ctx context.Context, body []byte, out interface{}) error {
	for _, v := range e.Hooks {
		if v.Decode != nil {
			return v.Decode(ctx, body, out)
		}
	}
	return errors.New("no decoder")
}

// Encode body
func (e Extension) Encode(ctx context.Context, body interface{}) ([]byte, error) {
	for _, v := range e.Hooks {
		if v.Encode != nil {
			return v.Encode(ctx, body)
		}
	}
	return nil, errors.New("no encoder")
}

// OnRequest process request
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

// RoundTrip process request to response
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

// OnResponse process response
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

// HandleOption process unknown options
func (e Extension) HandleOption(r *Request, o interface{}) (bool, error) {
	for _, v := range e.Hooks {
		if v.HandleOption != nil {
			if handle, err := v.HandleOption(r, o); err != nil {
				return false, err
			} else if handle {
				return true, nil
			}
		}
	}
	return false, nil
}
