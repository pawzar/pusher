.PHONY: help test build clean

help:
	@echo "============================================================"
	@echo "  Pusher make targets"
	@echo "============================================================"
	@egrep "^#" Makefile
	@echo "============================================================"

# test	- Run tests with coverage for `pusher` pakage
test:
	CGO_ENABLED=0 go test -v -cover ./pusher

# build	- Build the `push` binary
build: push

# clean	- Remove the compiled binary
clean:
	rm -f ./push

push:
	CGO_ENABLED=0 go build -o ./push main.go
