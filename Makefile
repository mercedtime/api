BIN=./mt
CMD=./cmd/mt

POSTGRES_DB ?= $$POSTGRES_DB
POSTGRES_USER ?= $$POSTGRES_USER
POSTGRES_PORT ?= $$POSTGRES_PORT
DUMP_FILE ?=

build:
	go build -o $(BIN) $(CMD)

clean:
	$(RM) coverage.txt
	if [ -x $(BIN) ]; then $(RM) $(BIN); fi
	if [ -x ./mtupdate ]; then $(RM) ./mtupdate; fi
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

dump:
	pg_dump -Fc -Z 9 \
		-h localhost -p $(POSTGRES_PORT) \
		--file=$(shell date +%m_%d_%y_%R)-database.dump \
		$(POSTGRES_DB) -U $(POSTGRES_USER)
	@if [ -f db/data/enrollment.dump ]; then rm db/data/enrollment.dump; fi
	pg_dump \
		-Fc -Z 9 \
		--data-only --table=enrollment \
		--file=db/data/enrollment.dump \
		-h localhost -p $(POSTGRES_PORT) \
		-d $(POSTGRES_DB) -U $(POSTGRES_USER)

%.dump:
	@echo $@

restore:
	@echo "wait stop"
	@#pg_restore -Fc -j $(shell nproc) $(DUMP_FILE) -d $(POSTGRES_DB) -U $(POSTGRES_USER)

.PHONY: build clean gen test coverage dump
