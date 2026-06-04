.PHONY: build run clean

BIN_DIR := bin
REPORT_DIR := reports

build:
	go build -o $(BIN_DIR)/llm-api-benchmark.exe ./main.go

run:
	go run ./main.go

clean:
	rm -rf $(BIN_DIR) $(REPORT_DIR)

test:
	go test ./...

.PHONY: build run clean test
