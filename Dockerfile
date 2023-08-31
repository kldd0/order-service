FROM golang:alpine

RUN apk add build-base

WORKDIR /app

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download

COPY ./ ./

RUN make build

CMD make bin-run
