BINARY := sandbox

.PHONY: build test integration vet install

build:
	go build -o $(BINARY) .

test:
	go test ./...

integration:
	SANDBOX_E2E=1 go test -tags=integration ./...

vet:
	go vet ./...

install: build
	install -Dm755 $(BINARY) $(HOME)/.local/bin/$(BINARY)
