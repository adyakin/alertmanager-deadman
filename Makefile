.PHONY: all clean

all: build

EXE = $(GOPATH)/bin/deadman
SRC = $(shell find ./ -type f -name '*.go')

$(EXE): $(SRC)
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-s -w -static"' .

build: $(EXE)

docker: deps
	docker build -t deadman .

clean:
	rm -f $(EXE)
