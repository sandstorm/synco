all: synco synco-lite
synco:
	mkdir -p build
	go build -o build/synco ./main.go
synco-lite:
	mkdir -p build
	go build -o build/synco-lite ./lite/main-lite.go

