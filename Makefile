EXECUTABLE=logrator
VERSION=$(shell git describe --tags --always --abbrev=0)


.PHONY: all test clean

all: test build

test:
	go test ./...

build: clean
	go build -v -o bin/$(EXECUTABLE) -ldflags="-s -w -X main.version=$(VERSION)" ./main.go


deb: build 
	VERSION=$(VERSION) nfpm pkg --packager deb --target ./bin/ -f ./packaging/nfpm.yml



clean:
	rm -f ./bin/*