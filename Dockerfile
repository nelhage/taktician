FROM golang:alpine
RUN apk update && apk add gcc libc-dev sqlite-dev

ADD . /go/src/github.com/nelhage/taktician/
WORKDIR /go/src/github.com/nelhage/taktician/

RUN go install -v github.com/nelhage/taktician/...
