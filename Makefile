check-module:
	@xcaddy list-modules

build-module:
	xcaddy build v2.10.0 --with github.com/devetek/caddyserver-minifier@main=./

run-jsonfile:
	./caddy run --config example/caddy.json

run-caddyfile:
	./caddy run --config example/Caddyfile

run-test:
	@go test -v -cover ./...


include scripts/makefile/*.mk