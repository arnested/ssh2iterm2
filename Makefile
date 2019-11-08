.PHONY: build

build: *.go
	go build -ldflags "$(shell govvv -flags)" -trimpath
