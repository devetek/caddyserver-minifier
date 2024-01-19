package minifier

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tdewolff/minify/v2"

	htmlminify "github.com/tdewolff/minify/v2/html"
	// cssminify "github.com/tdewolff/minify/v2/css"
	// jsminify "github.com/tdewolff/minify/v2/js"
)

type Middleware struct {
	minify *minify.M
	// content type minifier configuration
	Html htmlminify.Minifier
	// js   *jsminify.Minifier
	// css  *cssminify.Minifier
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.minifier",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// Provision implements caddy.Provisioner.
// https://pkg.go.dev/github.com/caddyserver/caddy/v2#Provisioner
// Invoke registered modules in the Caddy configuration
func (m *Middleware) Provision(ctx caddy.Context) error {
	if m.minify == nil {
		m.minify = minify.New()

		// set config minifier
		m.minify.Add("text/html", &m.Html)
	}

	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	fw := &responseMinifier{
		ResponseWriterWrapper: &caddyhttp.ResponseWriterWrapper{ResponseWriter: w},
		handler:               &m,
	}

	return next.ServeHTTP(fw, r)
}
