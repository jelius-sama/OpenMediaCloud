GOC := go

GOFLAGS_PROD := -ldflags "-s -w -X main.IS_PROD=TRUE  -X main.PORT=:8000" -trimpath -buildvcs=false
GOFLAGS_DEV  := -ldflags "-s -w -X main.IS_PROD=FALSE -X main.PORT=:8000" -trimpath -buildvcs=false

.PHONY: run build release

APP_NAME := OpenMediaCloud
BUILD := ./bin
ENTRY := ./cmd/

run:
	$(GOC) run $(GOFLAGS_DEV) $(ENTRY)

build:
	mkdir -p $(BUILD)
	$(GOC) build $(GOFLAGS_DEV) -o $(BUILD)/$(APP_NAME) $(ENTRY)

PLATFORMS = \
	linux-arm \
	linux-amd64 \
	linux-arm64 \
	linux-ppc64 \
	linux-ppc64le \
	linux-mips \
	linux-mipsle \
	linux-mips64 \
	linux-mips64le \
	linux-s390x \
	darwin-amd64 \
	darwin-arm64 \
	freebsd-amd64 \
	freebsd-386 \
	openbsd-amd64 \
	openbsd-arm64 \
	openbsd-386 \
	netbsd-arm \
	netbsd-amd64 \
	netbsd-386 \
	dragonfly-amd64 \
	solaris-amd64 \

release: $(PLATFORMS)

$(PLATFORMS):
	mkdir -p $(BUILD)
	GOOS=$(word 1,$(subst -, ,$@)) \
	GOARCH=$(word 2,$(subst -, ,$@)) \
	$(GOC) build $(GOFLAGS_PROD) \
	-o $(BUILD)/$(APP_NAME)-$@ $(ENTRY)
