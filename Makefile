.PHONY: build run test clean fmt vet lint check tidy install-ext

BINARY := vyr

build:
	go build -o $(BINARY) ./cmd/vyr

run:
	go run ./cmd/vyr $(ARGS)

test:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(BINARY) coverage.out coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

check: fmt vet test

install-ext:
	ln -sfn $(CURDIR)/editors/vscode ~/.vscode/extensions/nickyhof.vyr-0.1.0
	@echo "Installed to VS Code."
	@if [ -d ~/.antigravity/extensions ]; then \
		ln -sfn $(CURDIR)/editors/vscode ~/.antigravity/extensions/nickyhof.vyr-0.1.0; \
		echo "Installed to Antigravity."; \
	fi
	@echo "Reload your editor to activate."
