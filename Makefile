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


test:
	CGO_ENABLED=0 go test -v ./test_e2e/flowframework_test.go
