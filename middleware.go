package minifier

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tdewolff/minify/v2"
	htmlminify "github.com/tdewolff/minify/v2/html"
	svgminify "github.com/tdewolff/minify/v2/svg"
)

type Middleware struct {
	minify *minify.M
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

		// minifier config
		m.minify.Add("text/html", &htmlminify.Minifier{
			KeepWhitespace:      false,
			KeepDefaultAttrVals: true,
			KeepDocumentTags:    true,
			KeepEndTags:         true,
			KeepQuotes:          true,
		})
		m.minify.Add("image/svg+xml", &svgminify.Minifier{
			KeepComments: false,
		})
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
