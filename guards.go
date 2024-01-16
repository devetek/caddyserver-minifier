package minifier

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// Guard is list of active interface in Caddy lifecycle
var (
	_ caddy.Provisioner = (*Middleware)(nil)
	// _ caddy.Validator             = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
	_ caddyfile.Unmarshaler       = (*Middleware)(nil)

	// work with HTTP response
	_ http.ResponseWriter = (*responseMinifier)(nil)
)
