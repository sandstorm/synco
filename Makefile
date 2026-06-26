all: synco synco-lite
	#upx build/synco
	#upx --best --lzma build/synco-lite

# CGO_ENABLED=0
# -ldflags="-s -w" -> shrink the build
synco:
	mkdir -p build
	go build -ldflags="-s -w" -o build/synco ./main.go
synco-lite:
	mkdir -p build
	go build -ldflags="-s -w" -o build/synco-lite ./lite/main-lite.go


# run the full test suite, including the end-to-end tests (these require a
# running Docker daemon, as they spin up a MariaDB container via gnomock).
test:
	CGO_ENABLED=0 go test ./...

# run only the unit tests (no Docker required).
test-unit:
	CGO_ENABLED=0 go test $(shell go list ./... | grep -v /test_e2e)
