package req_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wenerme/go-req"
)

const httpbinURL = "https://httpbin.org"

// base request with httpbin + JSON codec
func httpbinReq() req.Request {
	return req.Request{
		BaseURL: httpbinURL,
		Options: []any{req.JSONEncode, req.JSONDecode},
	}
}

func TestHTTPBin_GET(t *testing.T) {
	var out struct {
		Args    map[string]string `json:"args"`
		Headers map[string]string `json:"headers"`
		URL     string            `json:"url"`
	}
	err := httpbinReq().With(req.Request{
		URL:   "/get",
		Query: map[string]string{"foo": "bar", "hello": "world"},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Args["foo"])
	assert.Equal(t, "world", out.Args["hello"])
	assert.Contains(t, out.URL, "httpbin.org/get")
}

func TestHTTPBin_POST_JSON(t *testing.T) {
	type Payload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	var out struct {
		JSON Payload `json:"json"`
		Data string  `json:"data"`
	}
	err := httpbinReq().With(req.Request{
		Method: "POST",
		URL:    "/post",
		Body:   Payload{Name: "test", Value: 42},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "test", out.JSON.Name)
	assert.Equal(t, 42, out.JSON.Value)
}

func TestHTTPBin_POST_Form(t *testing.T) {
	var out struct {
		Form map[string]string `json:"form"`
	}
	err := req.Request{
		BaseURL: httpbinURL,
		Method:  "POST",
		URL:     "/post",
		Body:    map[string]string{"key1": "val1", "key2": "val2"},
		Options: []any{req.FormEncode, req.JSONDecode},
	}.Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "val1", out.Form["key1"])
	assert.Equal(t, "val2", out.Form["key2"])
}

func TestHTTPBin_Headers(t *testing.T) {
	var out struct {
		Headers map[string]string `json:"headers"`
	}
	err := httpbinReq().With(req.Request{
		URL: "/headers",
		Header: http.Header{
			"X-Custom-Header": {"my-value"},
			"X-Another":       {"123"},
		},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "my-value", out.Headers["X-Custom-Header"])
	assert.Equal(t, "123", out.Headers["X-Another"])
}

func TestHTTPBin_StatusCodes(t *testing.T) {
	// 200
	resp, err := httpbinReq().With(req.Request{URL: "/status/200"}).Do()
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	// 404
	resp, err = httpbinReq().With(req.Request{URL: "/status/404"}).Do()
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	resp.Body.Close()

	// 500
	resp, err = httpbinReq().With(req.Request{URL: "/status/500"}).Do()
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	resp.Body.Close()
}

func TestHTTPBin_FetchBytes(t *testing.T) {
	body, resp, err := httpbinReq().With(req.Request{URL: "/bytes/128"}).FetchBytes()
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 128, len(body))
}

func TestHTTPBin_FetchString(t *testing.T) {
	body, resp, err := httpbinReq().With(req.Request{URL: "/html"}).FetchString()
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, body, "<html>")
}

func TestHTTPBin_PUT_DELETE_PATCH(t *testing.T) {
	for _, method := range []string{"PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			var out struct {
				Method string `json:"method"` // not returned by httpbin, but url shows method
				URL    string `json:"url"`
			}
			err := httpbinReq().With(req.Request{
				Method: method,
				URL:    "/" + strings.ToLower(method),
			}).Fetch(&out)
			require.NoError(t, err)
			assert.Contains(t, out.URL, strings.ToLower(method))
		})
	}
}

func TestHTTPBin_QueryMerge(t *testing.T) {
	// Base request has query, With() adds more
	base := httpbinReq().With(req.Request{
		URL:   "/get",
		Query: map[string]string{"base": "1"},
	})
	var out struct {
		Args map[string]string `json:"args"`
	}
	err := base.With(req.Request{
		Query: map[string]string{"extra": "2"},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "1", out.Args["base"])
	assert.Equal(t, "2", out.Args["extra"])
}

func TestHTTPBin_StructQuery(t *testing.T) {
	type Params struct {
		Page  int    `json:"page"`
		Size  int    `json:"size"`
		Query string `json:"query"`
	}
	var out struct {
		Args map[string]string `json:"args"`
	}
	err := httpbinReq().With(req.Request{
		URL:   "/get",
		Query: Params{Page: 1, Size: 20, Query: "hello world"},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, "1", out.Args["page"])
	assert.Equal(t, "20", out.Args["size"])
	assert.Equal(t, "hello world", out.Args["query"])
}

func TestHTTPBin_Context(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should timeout (httpbin /delay/5 waits 5 seconds)
	_, err := httpbinReq().With(req.Request{
		URL:     "/delay/5",
		Context: ctx,
	}).Do()
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestHTTPBin_ResponseHeaders(t *testing.T) {
	var resp *http.Response
	err := httpbinReq().With(req.Request{
		URL:   "/response-headers",
		Query: map[string]string{"X-Test": "hello"},
	}).Fetch(&resp)
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Header.Get("X-Test"))
}

func TestHTTPBin_Multipart(t *testing.T) {
	var out struct {
		Files map[string]string `json:"files"`
		Form  map[string]string `json:"form"`
	}

	content := "hello world file content"
	file := &req.MultipartFile{
		FieldName: "upload",
		Filename:  "test.txt",
		Reader:    strings.NewReader(content),
		Fields:    map[string]string{"description": "test upload"},
	}

	err := req.Request{
		BaseURL: httpbinURL,
		Method:  "POST",
		URL:     "/post",
		Body:    file,
		Options: []any{req.MultipartFormEncode, req.JSONDecode},
	}.Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, content, out.Files["upload"])
	assert.Equal(t, "test upload", out.Form["description"])
}

func TestHTTPBin_Hooks(t *testing.T) {
	var requestSeen, responseSeen bool

	hook := req.Hook{
		Name: "test-hook",
		OnRequest: func(r *http.Request) error {
			requestSeen = true
			r.Header.Set("X-Hook-Test", "yes")
			return nil
		},
		OnResponse: func(r *http.Response) error {
			responseSeen = true
			return nil
		},
	}

	var out struct {
		Headers map[string]string `json:"headers"`
	}
	err := httpbinReq().With(req.Request{
		URL:     "/headers",
		Options: []any{hook},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.True(t, requestSeen)
	assert.True(t, responseSeen)
	assert.Equal(t, "yes", out.Headers["X-Hook-Test"])
}

func TestHTTPBin_RawBody(t *testing.T) {
	payload := `{"raw": true, "value": 123}`
	var out struct {
		Data string         `json:"data"`
		JSON map[string]any `json:"json"`
	}
	err := req.Request{
		BaseURL: httpbinURL,
		Method:  "POST",
		URL:     "/post",
		RawBody: []byte(payload),
		Header:  http.Header{"Content-Type": {"application/json"}},
		Options: []any{req.JSONDecode},
	}.Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, payload, out.Data)
}

func TestHTTPBin_CustomRoundTripper(t *testing.T) {
	// Track that our custom transport was used
	var transportUsed bool
	customRT := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		transportUsed = true
		return http.DefaultTransport.RoundTrip(r)
	})

	var out struct {
		URL string `json:"url"`
	}
	err := httpbinReq().With(req.Request{
		URL:     "/get",
		Options: []any{req.UseRoundTripper(customRT)},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.True(t, transportUsed)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHTTPBin_DebugHook(t *testing.T) {
	var buf bytes.Buffer
	debug := req.DebugHook(&req.DebugOptions{
		Body: true,
		Out:  &buf,
	})
	err := httpbinReq().With(req.Request{
		URL:     "/get",
		Options: []any{debug},
	}).Fetch(&json.RawMessage{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "GET")
	assert.Contains(t, buf.String(), "httpbin.org")
}

func TestHTTPBin_WithComposition(t *testing.T) {
	// Test the declarative composition pattern used by go-wecom
	baseClient := httpbinReq()

	// Simulate "API method" pattern
	getIP := func() (string, error) {
		var out struct {
			Origin string `json:"origin"`
		}
		err := baseClient.With(req.Request{URL: "/ip"}).Fetch(&out)
		return out.Origin, err
	}

	getUserAgent := func() (string, error) {
		var out struct {
			UserAgent string `json:"user-agent"`
		}
		err := baseClient.With(req.Request{URL: "/user-agent"}).Fetch(&out)
		return out.UserAgent, err
	}

	ip, err := getIP()
	require.NoError(t, err)
	assert.NotEmpty(t, ip)

	ua, err := getUserAgent()
	require.NoError(t, err)
	assert.Contains(t, ua, "Go-http-client")
}

func TestHTTPBin_WithTimeout(t *testing.T) {
	// Should succeed — 200ms is enough for /get
	var out struct {
		URL string `json:"url"`
	}
	err := httpbinReq().With(req.Request{
		URL:     "/get",
		Options: []any{req.WithTimeout(5 * time.Second)},
	}).Fetch(&out)
	require.NoError(t, err)
	assert.Contains(t, out.URL, "httpbin.org")

	// Should timeout — 100ms is too short for /delay/3
	_, err = httpbinReq().With(req.Request{
		URL:     "/delay/3",
		Options: []any{req.WithTimeout(100 * time.Millisecond)},
	}).Do()
	require.Error(t, err)
}

func TestHTTPBin_FailOnStatus(t *testing.T) {
	// 200 should pass
	resp, err := httpbinReq().With(req.Request{
		URL:     "/status/200",
		Options: []any{req.FailOnStatus(req.IsHTTPError)},
	}).Do()
	require.NoError(t, err)
	resp.Body.Close()

	// 404 should fail
	_, err = httpbinReq().With(req.Request{
		URL:     "/status/404",
		Options: []any{req.FailOnStatus(req.IsHTTPError)},
	}).Do()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")

	// 500 with IsServerError
	_, err = httpbinReq().With(req.Request{
		URL:     "/status/500",
		Options: []any{req.FailOnStatus(req.IsServerError)},
	}).Do()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")

	// 400 should pass IsServerError (only checks >= 500)
	resp, err = httpbinReq().With(req.Request{
		URL:     "/status/400",
		Options: []any{req.FailOnStatus(req.IsServerError)},
	}).Do()
	require.NoError(t, err)
	resp.Body.Close()
}

func TestHTTPBin_GetBody(t *testing.T) {
	content := []byte(`{"streaming": true}`)
	var out struct {
		JSON map[string]any `json:"json"`
	}
	err := req.Request{
		BaseURL: httpbinURL,
		Method:  "POST",
		URL:     "/post",
		GetBody: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(content)), nil
		},
		Header:  http.Header{"Content-Type": {"application/json"}},
		Options: []any{req.JSONDecode},
	}.Fetch(&out)
	require.NoError(t, err)
	assert.Equal(t, true, out.JSON["streaming"])
}
