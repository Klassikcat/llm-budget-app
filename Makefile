.PHONY: test build build/tui build/gui build/gui-linux build/gui-macos run/gui-dev lint clean

WAILS ?= go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
WAILS_BUILD_FLAGS ?= -m -nosyncgomod
WAILS_LINUX_PLATFORM ?= linux/amd64
WAILS_MACOS_PLATFORM ?= darwin/universal

test:
	go test ./...

build: build/gui

build/tui:
	go build ./cmd/tui

build/gui: build/gui-linux

build/gui-linux:
	@if pkg-config --exists webkit2gtk-4.1; then \
		$(WAILS) build $(WAILS_BUILD_FLAGS) -platform $(WAILS_LINUX_PLATFORM) -tags webkit2_41; \
	elif pkg-config --exists webkit2gtk-4.0; then \
		$(WAILS) build $(WAILS_BUILD_FLAGS) -platform $(WAILS_LINUX_PLATFORM); \
	elif grep -q 'Ubuntu 24\.04' /etc/os-release 2>/dev/null; then \
		printf '%s\n' 'Missing WebKitGTK development headers. Install: sudo apt install libwebkit2gtk-4.1-dev' >&2; \
		exit 1; \
	else \
		printf '%s\n' 'Missing WebKitGTK development headers. Install either webkit2gtk-4.0 development headers, or webkit2gtk-4.1 development headers and build with the webkit2_41 tag.' >&2; \
		exit 1; \
	fi

build/gui-macos:
	$(WAILS) build $(WAILS_BUILD_FLAGS) -platform $(WAILS_MACOS_PLATFORM)

clean:
	@rm -f ./gui.bin ./llm-budget-tracker-gui ./tui
	@rm -rf ./build/bin

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
