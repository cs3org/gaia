all: build

service/gaia-frontend/node_modules:
	npm --prefix ./service/gaia-frontend install

service/static: service/gaia-frontend/node_modules
	npm --prefix ./service/gaia-frontend run build -- -o ../static -b /gaia/

.PHONY: build
build: service/static
	@go build

clean:
	@rm -f ./gaia && rm -rf ./service/static && rm -rf ./service/gaia-frontend/node_modules
