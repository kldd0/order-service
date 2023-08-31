CURDIR=$(shell pwd)
BINDIR=${CURDIR}/bin
LINTVER=v1.51.0
LINTBIN=${BINDIR}/lint_${GOVER}_${LINTVER}
MOCKGEN=${BINDIR}/mockgen_${GOVER}
PACKAGE=test-task/order-service/cmd/order-service

all: format build test lint

build: bindir
	go build -o ${BINDIR}/app ${PACKAGE}

test:
	go test ./...

run:
	go run ${PACKAGE}

bin-run:
	./bin/app

lint: install-lint
	${LINTBIN} run

bindir:
	mkdir -p ${BINDIR}

generate: install-mockgen
	${MOCKGEN} -source=internal/http-server/handlers/order/get/get.go -destination=internal/http-server/handlers/order/get/mocks/order_getter.go
	${MOCKGEN} -source=internal/cache/cache.go -destination=internal/cache/mocks/cache_mock.go
	# ${MOCKGEN} -source=internal/database/database.go -destination=internal/mocks/database/database_mocks.go

format:
	go fmt ./...

vet:
	go vet ./...

precommit: format build test lint
	echo "OK"

dc:
	docker-compose up --remove-orphans --build

dcd:
	docker-compose up --remove-orphans --build -d

install-lint: bindir
	test -f ${LINTBIN} || \
		(GOBIN=${BINDIR} go install github.com/golangci/golangci-lint/cmd/golangci-lint@${LINTVER} && \
		mv ${BINDIR}/golangci-lint ${LINTBIN})

install-mockgen: bindir
	test -f ${MOCKGEN} || \
		(GOBIN=${BINDIR} go install github.com/golang/mock/mockgen@v1.6.0 && \
		mv ${BINDIR}/mockgen ${MOCKGEN})
