.PHONY: clean build

clean:
	rm -rf $(ARTIFACTS_DIR)

build:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(ARTIFACTS_DIR)/bootstrap