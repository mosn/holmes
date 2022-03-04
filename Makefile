lint:
	golangci-lint run --tests=false --skip-dirs=example
test:
	GO111MODULE=on go test -gcflags=-l -v
