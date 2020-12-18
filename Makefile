BIN=./mt
CMD=./cmd/mt

POSTGRES_DB ?= $$POSTGRES_DB
POSTGRES_USER ?= $$POSTGRES_USER
POSTGRES_PORT ?= $$POSTGRES_PORT
PG_HOST ?= localhost
DUMP_FILE ?= ./full-database.dump

ENROLLMENT_DUMP ?= db/data/spring-2021/enrollment.dump

build:
	go build -o $(BIN) $(CMD)

clean:
	$(RM) coverage.txt
	if [ -x $(BIN) ]; then $(RM) $(BIN); fi
	if [ -x ./mtupdate ]; then $(RM) ./mtupdate; fi
	#$(RM) db/data/*.csv *.test

gen:
	$(RM) db/data/spring-2021/*.csv
	go generate ./...

test:
	@env $(shell cat ../.env) go test ./... \
		-cover -coverprofile=coverage.txt   \
		-covermode=atomic

coverage:
	go tool cover -html=coverage.txt

build-test-image:
	docker image build -t mt-api.tests . -f ./docker/Dockerfile.tests
run-test-image:
	docker container run --rm -it mt-api.test

dump:
	psql -h $(PG_HOST) -p $(POSTGRES_PORT) -d $(POSTGRES_DB) -U $(POSTGRES_USER) -c 'select * from counts'
	@if [ -f $(DUMP_FILE) ]; then rm $(DUMP_FILE); fi
	pg_dump -Fc -Z 9 \
		-h $(PG_HOST) -p $(POSTGRES_PORT) \
		--file=$(DUMP_FILE) \
		-d $(POSTGRES_DB) -U $(POSTGRES_USER)

	@if [ -f $(ENROLLMENT_DUMP) ]; then rm $(ENROLLMENT_DUMP); fi
	pg_dump \
		-Fc -Z 9 \
		--data-only --table=enrollment \
		--file=$(ENROLLMENT_DUMP) \
		-h $(PG_HOST) -p $(POSTGRES_PORT) \
		-d $(POSTGRES_DB) -U $(POSTGRES_USER)

db/data/mercedtime.dump:
	@if [ -f db/data/mercedtime.dump ]; then rm db/data/mercedtime.dump; fi
	pg_dump -Fc -Z 9 \
		-h localhost -p $(POSTGRES_PORT) \
		--file=db/data/mercedtime.dump \
		$(POSTGRES_DB) -U $(POSTGRES_USER)

restore:
	@echo "wait stop"
	@#pg_restore -Fc -j $(shell nproc) $(DUMP_FILE) -d $(POSTGRES_DB) -U $(POSTGRES_USER)

historical-data:
	$(RM) db/data/fall-2020/*.csv db/data/summer-2020/*.csv
	go run ./cmd/mtupdate -csv -out=db/data/fall-2020 -year=2020 -term=fall
	go run ./cmd/mtupdate -csv -out=db/data/summer-2020 -year=2020 -term=summer


.PHONY: build clean gen test coverage dump
