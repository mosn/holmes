lint:
	golangci-lint run --timeout=10m --tests=false --skip-dirs=example
test:
	GO111MODULE=on go test -gcflags=-l -v
