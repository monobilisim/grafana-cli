.PHONY: build clean install

build:
	CGO_ENABLED=0 go build -ldflags="-extldflags=-static" -o gcli .

install: build
	sudo mv gcli /usr/local/bin/gcli
	@echo "gcli installed to /usr/local/bin/gcli"
	@gcli completion install

uninstall:
	-@gcli completion uninstall
	sudo rm -f /usr/local/bin/gcli
	@echo "gcli uninstalled from /usr/local/bin/gcli"

clean:
	rm -f gcli
