
ci: lint test

fmt:
	gofumpt -w `find . -type f -name '*.go' -not -path "./vendor/*"`
	goimports -w `find . -type f -name '*.go' -not -path "./vendor/*"`

lint:
	golangci-lint run

test:
	go test ./...
