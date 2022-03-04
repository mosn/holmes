modules=$(shell go list ./... | grep -v example)
test:
	GO111MODULE=on go test -gcflags=-l -v $(modules)
