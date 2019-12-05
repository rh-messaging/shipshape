.PHONY: all
all: build

.PHONY: format
format:
	go fmt ./pkg/...

.PHONY: build
build:
	go build ./pkg/...

.PHONY: cluster-test
cluster-test:
	go build ./test/...
	go test --count=1 -v "./test/framework"
