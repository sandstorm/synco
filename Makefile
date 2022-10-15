all: synco synco-source
synco:
	mkdir -p build
	go build -o build/synco ./main.go
synco-source:
	mkdir -p build
	go build -o build/synco-source ./main-source.go

