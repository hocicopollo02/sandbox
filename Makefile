BINARY := sandbox

.PHONY: build test vet install

build:
	go build -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

install: build
	install -Dm755 $(BINARY) $(HOME)/.local/bin/$(BINARY)
