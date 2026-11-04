GO ?= go
BINARY ?= blackhole-threats
CMD_PATH ?= ./cmd/$(BINARY)
PKG ?= github.com/netspeedy/blackhole-threats
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || (dpkg-parsechangelog -SVersion 2>/dev/null | sed 's/-[^-]*$$//') || printf '%s' dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf '%s' packaged)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOFLAGS ?= -trimpath
SITE_URL ?= https://netspeedy.github.io/blackhole-threats/
SITE_RELEASE_VERSION ?= $(VERSION)
SITE_COMMIT ?= $(COMMIT)
SITE_BUILD_DATE ?= $(BUILD_DATE)
SITE_APT_FINGERPRINT ?= Published alongside stable release builds.
WEBSITE_DIR ?= website
LDFLAGS ?= -s -w \
	-X $(PKG)/internal/buildinfo.Version=$(VERSION) \
	-X $(PKG)/internal/buildinfo.Commit=$(COMMIT) \
	-X $(PKG)/internal/buildinfo.BuildDate=$(BUILD_DATE)

.PHONY: fmt fmt-check vet test build build-cross docker-build website-build readme-hero clean package

test:
	$(GO) test ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

vet:
	$(GO) vet ./...

build:
	mkdir -p dist
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY) $(CMD_PATH)

build-cross:
	test -n "$(GOOS)"
	test -n "$(GOARCH)"
	test -n "$(OUTPUT)"
	mkdir -p $$(dirname "$(OUTPUT)")
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(OUTPUT) $(CMD_PATH)

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(BINARY):dev \
		.

website-build:
	PUBLIC_SITE_URL=$(SITE_URL) \
	PUBLIC_RELEASE_VERSION=$(SITE_RELEASE_VERSION) \
	PUBLIC_COMMIT=$(SITE_COMMIT) \
	PUBLIC_BUILD_DATE=$(SITE_BUILD_DATE) \
	PUBLIC_APT_FINGERPRINT='$(SITE_APT_FINGERPRINT)' \
	npm --prefix $(WEBSITE_DIR) run build

readme-hero:
	uv run scripts/readme/build_readme_hero.py

clean:
	rm -rf _build dist

package:
	BLACKHOLE_LDFLAGS="$(LDFLAGS)" dpkg-buildpackage -us -uc -b
