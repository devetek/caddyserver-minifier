check-module:
	@xcaddy list-modules

run-module:
	xcaddy run -c example/caddy.json

run-module-caddy:
	xcaddy run -c example/Caddyfile

build-module:
	xcaddy build v2.7.6 --with github.com/devetek/caddyserver-minifier@main=./

run-build-module:
	./caddy run -c example/caddy.json

run-test:
	@go test -v -cover ./...


include scripts/makefile/*.mk