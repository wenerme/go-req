package req_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wenerme/go-req"

	"github.com/stretchr/testify/assert"
)

func TestOverride(t *testing.T) {
	{
		assert.Equal(t,
			req.Request{}.With(req.Request{Values: req.Values{}.Add("1", "2").Add("1", "3")}),
			req.Request{Values: req.Values{}.Add("1", "2")}.With(req.Request{Values: req.Values{}.Add("1", "3")}),
		)
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

func TestHookPreserve(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("OK"))
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
		_, err := req.Request{
			BaseURL: server.URL,
			Options: []interface{}{
				req.DebugHook(&req.DebugOptions{
					Body: true,
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
	}
}
