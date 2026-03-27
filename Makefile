GOC := GOOS=linux GOARCH=arm64 go

GOFLAGS_PROD := -ldflags "-s -w -X main.IS_PROD=TRUE  -X main.PORT=:8000" -trimpath -buildvcs=false
GOFLAGS_DEV  := -ldflags "-s -w -X main.IS_PROD=FALSE -X main.PORT=:8000" -trimpath -buildvcs=false

.PHONY: run build release

APP_NAME := ClientToR2
BUILD := ./bin
ENTRY := ./cmd/

run:
	$(GOC) run $(GOFLAGS_DEV) $(ENTRY)

build:
	mkdir -p $(BUILD)
	$(GOC) build $(GOFLAGS_DEV) -o $(BUILD)/$(APP_NAME) $(ENTRY)

release:
	mkdir -p $(BUILD)
	$(GOC) build $(GOFLAGS_PROD) -o $(BUILD)/$(APP_NAME) $(ENTRY)
