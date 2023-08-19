CURDIR=$(shell pwd)
BINDIR=${CURDIR}/bin
LINTVER=v1.51.0
LINTBIN=${BINDIR}/lint_${GOVER}_${LINTVER}
PACKAGE=test-task/order-service/cmd/order-service

all: format build test lint

build: bindir
	go build -o ${BINDIR}/app ${PACKAGE}

test:
	go test ./...

run:
	go run ${PACKAGE}

lint: install-lint
	${LINTBIN} run

bindir:
	mkdir -p ${BINDIR}

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
