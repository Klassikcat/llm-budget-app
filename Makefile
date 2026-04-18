.PHONY: test build-tui build-gui run-gui-dev lint

test:
	go test ./...

build/tui:
	go build ./cmd/tui

build/gui:
	@if pkg-config --exists webkit2gtk-4.1; then \
		go build -tags desktop,production,webkit2_41 -o gui.bin ./cmd/gui; \
	elif pkg-config --exists webkit2gtk-4.0; then \
		go build -tags desktop,production -o gui.bin ./cmd/gui; \
	elif grep -q 'Ubuntu 24\.04' /etc/os-release 2>/dev/null; then \
		printf '%s\n' 'Missing WebKitGTK development headers. Install: sudo apt install libwebkit2gtk-4.1-dev' >&2; \
		exit 1; \
	else \
		printf '%s\n' 'Missing WebKitGTK development headers. Install either webkit2gtk-4.0 development headers, or webkit2gtk-4.1 development headers and build with the webkit2_41 tag.' >&2; \
		exit 1; \
	fi

clean:
	@rm -f ./gui.bin ./tui

run/gui-dev:
	@if pkg-config --exists webkit2gtk-4.1; then \
		go run -tags dev,webkit2_41 ./cmd/gui; \
	elif pkg-config --exists webkit2gtk-4.0; then \
		go run -tags dev ./cmd/gui; \
	elif grep -q 'Ubuntu 24\.04' /etc/os-release 2>/dev/null; then \
		printf '%s\n' 'Missing WebKitGTK development headers. Install: sudo apt install libwebkit2gtk-4.1-dev' >&2; \
		exit 1; \
	else \
		printf '%s\n' 'Missing WebKitGTK development headers. Install either webkit2gtk-4.0 development headers, or webkit2gtk-4.1 development headers and rebuild with the webkit2_41 tag.' >&2; \
		exit 1; \
	fi

lint:
	@printf '%s\n' 'lint placeholder'
