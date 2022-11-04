.PHONY: build test

build:
	go build -o conduit-connector-log cmd/connector/main.go

test:
	go test $(GOTEST_FLAGS) -v -race ./...

