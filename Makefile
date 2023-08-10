all: build

.PHONY: build
build: 
	@go build

clean:
	@rm -f ./gaia
