all: synco synco-source
synco:
	mkdir -p build
	go build  -o build/synco ./cmd/synco/
synco-source:
	mkdir -p build
	go build  -o build/synco-source ./cmd/synco-source/

