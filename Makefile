.PHONY:

default: build

clean:
	rm -Rf bin/*

build:
	mkdir -p bin/
	go build -o bin/summit cmd/summit/summit.go
