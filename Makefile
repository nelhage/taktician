PREFIX := github.com/nelhage/taktician

build:
	go build $(PREFIX)/...

install:
	go install $(PREFIX)/...

test:
	go test $(PREFIX)/...

test-%:
	go test $(PREFIX)/$*...

.PHONY: test install build
