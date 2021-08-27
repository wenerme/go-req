# go-req

[![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]  [![Go Report Card][report-card-img]][report-card]

[doc-img]: https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square
[doc]: https://pkg.go.dev/github.com/wenerme/go-req?tab=doc
[ci-img]: https://github.com/wenerme/go-req/actions/workflows/ci.yml/badge.svg
[ci]: https://github.com/wenerme/go-req/actions/workflows/ci.yml
[cov-img]: https://codecov.io/gh/wenerme/go-req/branch/main/graph/badge.svg
[cov]: https://codecov.io/gh/wenerme/go-req/branch/main
[report-card-img]: https://goreportcard.com/badge/github.com/wenerme/go-req
[report-card]: https://goreportcard.com/report/github.com/wenerme/go-req

Declarative golang HTTP client

```go
package req_test

import (
    "fmt"
    "github.com/wenerme/go-req"
    "net/http"
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
        URL: "/post",
        Body: HelloRequest{
          Name: "go-req",
        },
    }).Fetch(&out,&r)
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
```

## Used by

- [wenerme/go-wecom](https://github.com/wenerme/go-wecom)
  - Wechat Work/Wecom/企业微信 Golang SDK
