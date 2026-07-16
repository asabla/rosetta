.PHONY: build check fuzz test

build:
	go build ./cmd/...

test:
	go test ./...

fuzz:
	go test . -run='^$$' -fuzz=FuzzCheckNeverPanics -fuzztime=30s
	go test . -run='^$$' -fuzz=FuzzCompileNeverBroadensDeniedInput -fuzztime=30s
	go test . -run='^$$' -fuzz=FuzzGlobIntersectionNeverMissesWitness -fuzztime=30s

check:
	sh scripts/check-toolchain.sh
	test -z "$$(gofmt -l .)"
	go test ./...
	go test -race ./...
	go vet ./...
	go build ./cmd/...
