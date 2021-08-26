package req_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wenerme/go-req"

	"github.com/stretchr/testify/assert"
)

func TestOverride(t *testing.T) {
	assert.Equal(t, req.Request{
		Method:  "POST",
		BaseURL: "http//wener.me",
		Context: context.Background(),
		Query: map[string][]string{
			"a": {"a"},
			"b": {"b"},
		},
		RawBody: []byte("HELLO"),
	}, req.Request{
		Method: "GET",
		Query: url.Values{
			"a": []string{"a"},
		},
	}.With(req.Request{
		Method:  "POST",
		BaseURL: "http//wener.me",
		RawBody: []byte("HELLO"),
		Query: url.Values{
			"b": []string{"b"},
		},
		Context: context.Background(),
	}))
}

func TestReconcile(t *testing.T) {
	{
		r := req.Request{}
		assert.NoError(t, r.Reconcile())
		assert.Equal(t, req.Request{
			Method:  "GET",
			Context: context.Background(),
		}, r)
	}
	{
		// support Request as option
		r := req.Request{
			Options: []interface{}{
				nil,
				req.Request{Method: "POST"},
			},
		}
		assert.NoError(t, r.Reconcile())
		assert.Equal(t, req.Request{
			Method:  "POST",
			Context: context.Background(),
		}, r)
	}
}

func TestReconcileHandleOption(t *testing.T) {
	cnt := 0
	r := req.Request{}
	{
		rr := r.With(req.Request{
			Options: []interface{}{1},
		})
		assert.Error(t, rr.Reconcile())
	}
	r.Extension.With(req.Hook{HandleOption: func(r *req.Request, o interface{}) (bool, error) {
		v, ok := o.(int)
		if !ok {
			return false, nil
		}
		cnt += v
		return true, nil
	}})
	r = r.With(req.Request{
		Options: []interface{}{1},
	})
	assert.NoError(t, r.Reconcile())
	assert.Equal(t, 1, cnt)
}

func TestReconcileOptionsOrder(t *testing.T) {
	cnt := 0
	r := req.Request{
		Options: []interface{}{
			func(r *req.Request) {
				assert.Equal(t, cnt, 2)
				cnt++
			},
			func(r *req.Request) {
				assert.Equal(t, cnt, 3)
				cnt++
			},
		},
	}
	r = r.With(req.Request{
		Options: []interface{}{
			func(r *req.Request) {
				// first call
				assert.Equal(t, cnt, 0)
				cnt++
			},
			func(r *req.Request) {
				// first call
				assert.Equal(t, cnt, 1)
				cnt++
			},
		},
	})
	assert.NoError(t, r.Reconcile())
	assert.Equal(t, 4, cnt)
}

func ExampleRequest() {
	var out HelloResponse
	err := req.Request{
		BaseURL: "https://example.com",
		URL:     "/hello",
		Body: HelloRequest{
			Name: "wener",
		},
		Options: []interface{}{req.JSONEncode, req.JSONDecode},
	}.Fetch(&out)
	if err != nil {
		panic(err)
	}
}

func TestUrlBuild(t *testing.T) {
	{
		r, err := req.Request{
			BaseURL: "https://wener.me",
			URL:     "/token",
		}.NewRequest()
		assert.NoError(t, err)
		assert.Equal(t, "https://wener.me/token", r.URL.String())
	}
	{
		r, err := req.Request{
			BaseURL: "https://wener.me",
			URL:     "/token",
			Query: map[string][]string{
				"name": {"wener"},
			},
		}.NewRequest()
		assert.NoError(t, err)
		assert.Equal(t, "https://wener.me/token?name=wener", r.URL.String())
	}
	{
		r, err := req.Request{
			BaseURL: "https://wener.me",
			URL:     "/token",
			Query: map[string]string{
				"name": "wener",
			},
		}.NewRequest()
		assert.NoError(t, err)
		assert.Equal(t, "https://wener.me/token?name=wener", r.URL.String())
	}
	{
		r, err := req.Request{
			BaseURL: "https://wener.me",
			URL:     "/token",
			Query: map[string]interface{}{
				"name": "wener",
				"age":  18,
			},
		}.NewRequest()
		assert.NoError(t, err)
		assert.Equal(t, "https://wener.me/token?age=18&name=wener", r.URL.String())
	}
}

// nolint: funlen
func TestHookPreserve(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("OK"))
	})
	mux.HandleFunc("/echo", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.Copy(writer, request.Body)
	})
	mux.HandleFunc("/query", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.URL.RawQuery))
	})
	mux.HandleFunc("/echo/head", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.Header.Get("Echo")))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	{
		request, err := req.Request{
			BaseURL: "https://wener.me",
			Options: []interface{}{
				req.Hook{
					OnRequest: func(r *http.Request) error {
						r.Header.Set("Run", "true")
						return nil
					},
				},
			},
		}.NewRequest()
		assert.NoError(t, err)
		assert.Equal(t, "true", request.Header.Get("Run"))
	}

	{
		run := false
		empty := bytes.Buffer{}
		err := req.Request{
			BaseURL: server.URL,
			Options: []interface{}{
				req.DebugHook(nil),
				req.DebugHook(&req.DebugOptions{
					Body: true,
				}),
				req.DebugHook(&req.DebugOptions{
					Disable: true,
					Out:     &empty,
				}),
				req.Hook{
					OnResponse: func(r *http.Response) error {
						run = true
						return nil
					},
				},
			},
		}.Fetch()
		assert.NoError(t, err)
		assert.True(t, run)
		assert.Equal(t, 0, empty.Len())
	}

	r := req.Request{
		BaseURL: server.URL,
	}
	{
		var out HelloRequest
		var request *http.Request
		var response *http.Response
		err := r.With(req.Request{
			URL:  "/echo",
			Body: HelloRequest{Name: "wener"},
		}).WithHook(req.JSONDecode, req.JSONEncode).Fetch(&out, &request, &response)
		assert.NoError(t, err)
		assert.Equal(t, "wener", out.Name)
		assert.NotNil(t, request)
		assert.NotNil(t, response)
	}
	{
		out, _, err := r.With(req.Request{
			URL: "/echo",
			GetBody: func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("HELLO")), nil
			},
		}).WithHook(req.JSONDecode, req.JSONEncode).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, "HELLO", out)
	}
	{
		out, _, err := r.With(req.Request{
			URL:  "/echo",
			Body: HelloRequest{Name: "wener"},
		}).WithHook(req.JSONEncode).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `{"Name":"wener"}`, out)
	}
	{
		out, _, err := r.With(req.Request{
			URL:  "/echo",
			Body: HelloRequest{Name: "wener"},
		}).WithHook(req.FormEncode).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `Name=wener`, out)
	}
	{
		out, _, err := r.With(req.Request{
			Options: []interface{}{req.JSONEncode},
		}).With(req.Request{
			URL:  "/echo",
			Body: HelloRequest{Name: "wener"},
			// FormEncode is higher than JSONEncode
			Options: []interface{}{req.FormEncode},
		}).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `Name=wener`, out)
	}
	{
		// test header
		rr := r.With(req.Request{
			URL: "/echo/head",
			Header: map[string][]string{
				"echo": {"OK"},
			},
		})
		out, _, err := rr.FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `OK`, out)

		out, _, err = rr.With(req.Request{
			Options: []interface{}{req.JSONEncode},
			Header: map[string][]string{
				"echo": {"Override"},
			},
		}).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `Override`, out)
	}
	{
		out, _, err := r.With(req.Request{
			URL:   "/query",
			Query: HelloRequest{Name: "wener"},
		}).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `Name=wener`, out)
	}
	{
		out, _, err := r.With(req.Request{
			URL:      "/query",
			RawQuery: "a=1",
		}).FetchString()
		assert.NoError(t, err)
		assert.Equal(t, `a=1`, out)
	}
	{
		// test options
		_, _, err := r.With(req.Request{
			URL: "/query",
			Options: []interface{}{
				func(r *req.Request) {},
				func(r *req.Request) error {
					return nil
				},
			},
		}).FetchString()
		assert.NoError(t, err)
	}
	{
		// test option error
		_, _, err := r.With(req.Request{
			URL: "/query",
			Options: []interface{}{
				func(r *req.Request) error {
					return io.EOF
				},
			},
		}).FetchString()
		assert.Equal(t, io.EOF, err)
	}
	{
		// test invalid option
		_, _, err := r.With(req.Request{
			URL: "/query",
			Options: []interface{}{
				func() {},
			},
		}).FetchString()
		assert.Error(t, err)
	}
}

func TestRT(t *testing.T) {
	res, err := req.Request{
		URL: "http://wener.me",
	}.WithHook(req.UseRoundTripper(rtFunc(func(request *http.Request) (*http.Response, error) {
		return nil, nil
	}))).Do()
	assert.NoError(t, err)
	assert.Nil(t, res)
}

type HelloRequest struct {
	Name string
}

type HelloResponse struct {
	Hello string
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
