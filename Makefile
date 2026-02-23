# Digital Signage Makefile

.PHONY: build clean install test run dev cross-compile pi-zero pi-3 pi-4

# Build for current platform
.PHONY: build
build:
	go build -ldflags="-w -s" -o digital-signage .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf build/
	rm -f digital-signage

# Create build directory
build/:
	mkdir -p build

# Cross-compile for Raspberry Pi Zero (ARMv6)
.PHONY: pi-zero
pi-zero: build/
	env GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=${version}" -o build/digital-signage-pi-zero .

# Cross-compile for Raspberry Pi 3 (ARMv7)
.PHONY: pi-3
pi-3: build/
	env GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=${version}" -o build/digital-signage-pi-3 .

# Cross-compile for Raspberry Pi 4 (ARM64)
.PHONY: pi-4
pi-4: build/
	env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=${version}" -o build/digital-signage-pi-4 .

# Cross-compile for all Raspberry Pi targets
.PHONY: cross-compile
cross-compile: pi-zero pi-3 pi-4
	@echo "Cross-compilation completed!"
	@echo "Build artifacts:"
	@ls -la build/

# Run locally for development
.PHONY: run
run:
	go run .

# Run with race detection
.PHONY: dev
dev:
	go run -race .

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Create release package
release: cross-compile
	mkdir -p release
	tar -czf release/digital-signage-pi-zero.tar.gz -C build digital-signage-pi-zero
	tar -czf release/digital-signage-pi-3.tar.gz -C build digital-signage-pi-3
	tar -czf release/digital-signage-pi-4.tar.gz -C build digital-signage-pi-4
	cp digital-signage.service release/
	cp install.sh release/
	cp kiosk.sh release/
	cp .env.example release/
	@echo "Release packages created in release/ directory"

# Install on local Raspberry Pi (requires SSH setup)
install-pi: cross-compile
	@read -p "Enter Raspberry Pi IP address: " PI_IP; \
	scp build/digital-signage-pi-* pi@$$PI_IP:~/; \
	scp digital-signage.service install.sh kiosk.sh .env.example pi@$$PI_IP:~/; \
	ssh pi@$$PI_IP 'chmod +x install.sh kiosk.sh && ./install.sh'
