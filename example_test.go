package req_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wenerme/go-req"
)

func ExampleRequest() {
	// reusable client
	client := req.Request{
		BaseURL: "https://httpbin.org",
		Options: []interface{}{req.JSONEncode, req.JSONDecode},
	}

	// dump request and response with body
	client = client.WithHook(req.DebugHook(&req.DebugOptions{Body: true}))

	// send request with declarative override
	var out PostResponse
	var r *http.Response
	err := client.With(req.Request{
		Method: http.MethodPost,
		URL:    "/post",
		Body: HelloRequest{
			Name: "go-req",
		},
	}).Fetch(&out, &r)
	if err != nil {
		panic(err)
	}
	// print go-req
	fmt.Println(out.JSON.Name)
	// print 200
	fmt.Println(r.StatusCode)

	// override Options use form encode
	err = client.With(req.Request{
		Method: http.MethodPost,
		URL:    "/post",
		Body: HelloRequest{
			Name: "go-req",
		},
		Options: []interface{}{req.FormEncode},
	}).Fetch(&out)
	if err != nil {
		panic(err)
	}
	// print go-req
	fmt.Println(out.Form.Name)
}

type HelloRequest struct {
	Name string
}

type HelloResponse struct {
	Name string
}

type PostResponse struct {
	JSON HelloResponse `json:"json"`
	Form HelloResponse `json:"form"`
}

func TestRunExample(t *testing.T) {
	ExampleRequest()
}
