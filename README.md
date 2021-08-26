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
package main

import req "github.com/wenerme/go-req"

func main() {
	var out struct{ Hello string }
	_, err := req.Request{
		BaseURL: "https://example.com",
		URL:     "/hello",
		Body: struct {
			Name string
		}{Name: "wener"},
		Options: []interface{}{req.JSONEncode, req.JSONDecode},
	}.Fetch(&out)
	if err != nil {
		panic(err)
	}
}
```

## Used by

- [wenerme/go-wecom](https://github.com/wenerme/go-wecom)
  - Wechat Work/Wecom/企业微信 Golang SDK
