package req_test

import (
	"fmt"
	"github.com/wenerme/go-req"
	"net/http"
	"testing"
)

func ExampleRequest() {
	// reusable client
	client := req.Request{
		BaseURL: "https://httpbin.org",
		Options: []interface{}{req.JSONEncode, req.JSONDecode},
	}

	// dump request and response with body
	client = client.WithHook(req.DebugHook(&req.DebugOptions{Body: true}))

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
}

type HelloRequest struct {
	Name string
}
type HelloResponse struct {
	Name string
}
type PostResponse struct {
	JSON HelloResponse `json:"json"`
}

func TestRunExample(t *testing.T) {
	ExampleRequest()
}
