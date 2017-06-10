PREFIX := github.com/nelhage/taktician

PROTOS := $(wildcard proto/*.proto)
PROTONAMES := $(basename $(notdir $(PROTOS)))
GOPROTOSRC := $(foreach proto,$(PROTONAMES),pb/$(proto).pb.go)
PYPROTOSRC := $(foreach proto,$(PROTONAMES),python/tak/proto/$(proto)_pb2.py)
GENFILES := ai/feature_string.go $(GOPROTOSRC) $(PYPROTOSRC)


ai/feature_string.go: ai/evaluate.go
	go generate $(PREFIX)/ai

protoc: $(GOPROTOSRC) $(PYPROTOSRC)

$(GOPROTOSRC) $(PYPROTOSRC): $(PROTOS)
	protoc -I proto/ \
	       --python_out=python/tak/proto/ --go_out=pb \
	       proto/*.proto

build: $(GENFILES)
	go build $(PREFIX)/...

install: $(GENFILES)
	go install $(PREFIX)/...

test: $(GENFILES)
	go test $(PREFIX)/...

test-%: $(GENFILES)
	go test $(PREFIX)/$*...

.PHONY: test install build protoc
