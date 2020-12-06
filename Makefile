BIN=./api
build:
	go build -o $(BIN) ./cmd/api

clean:
	if [ -x $(BIN) ]; then $(RM) $(BIN); fi
	if [ -x ./mtupdate ]; then $(RM) ./mtupdate; fi

.PHONY: build clean
