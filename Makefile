BIN_DIR := bin
APP := gopick
OUT := $(BIN_DIR)/$(APP)

.PHONY: build clean

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(OUT) ./cmd/$(APP)

clean:
	rm -f $(OUT)
