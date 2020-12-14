BIN=./mt
CMD=./cmd/mt

build:
	go build -o $(BIN) $(CMD)

clean:
	$(RM) coverage.txt
	if [ -x $(BIN) ]; then $(RM) $(BIN); fi
	if [ -x ./mtupdate ]; then $(RM) ./mtupdate; fi
	if [ -x ./mt ]; then $(RM) ./mtupdate; fi
	$(RM) db/data/*.csv *.test

gen:
	go generate ./...

test:
	@env $(shell cat ../.env) go test ./... \
		-cover -coverprofile=coverage.txt \
		-covermode=atomic

coverage:
	go tool cover -html=coverage.txt

build-test-image:
	docker image build -t mt-api.tests . -f ./docker/Dockerfile.tests
run-test-image:
	docker container run --rm -it mt-api.test


.PHONY: build clean
