package req

import "context"

type contextKey string

func (c contextKey) String() string {
	return "req.contextKey(" + string(c) + ")"
}

const RequestContextKey = contextKey("Request")

// NewContext with Request
func NewContext(ctx context.Context, v *Request) context.Context {
	return context.WithValue(ctx, RequestContextKey, v)
}

// FromContext get Request from context.Context
func FromContext(ctx context.Context) *Request {
	v := ctx.Value(RequestContextKey)
	if v == nil {
		return nil
	}
	return v.(*Request)
}
