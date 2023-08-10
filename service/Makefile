all: build

gaia-frontend/node_modules:
	npm --prefix ./gaia-frontend install

static: gaia-frontend/node_modules
	npm --prefix ./gaia-frontend run build -- -o ../static -b /gaia/

.PHONY: build
build: static
	@go build -o gaiasvc cmd/gaiasvc/main.go

clean:
	@rm -f ./gaiasvc && rm -rf ./static && rm -rf ./gaia-frontend/node_modules