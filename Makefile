BINARY := sandbox

.PHONY: build test integration vet hooks install

build:
	go build -o $(BINARY) .

test:
	go test ./...

integration:
	SANDBOX_E2E=1 go test -tags=integration ./...

vet:
	go vet ./...

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks enabled from .githooks"

install: build
	install -Dm755 $(BINARY) $(HOME)/.local/bin/$(BINARY)
