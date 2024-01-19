package minifier

import (
	"log"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	htmlminify "github.com/tdewolff/minify/v2/html"
)

var moduleName = "minifier"

func init() {
	caddy.RegisterModule(Middleware{})
	httpcaddyfile.RegisterHandlerDirective(moduleName, parseCaddyfile)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	m := new(Middleware)
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

func converterStrToBoolean(str string) bool {
	boolValue, err := strconv.ParseBool(str)
	if err != nil {
		// print error info
		log.Println("error on converterStrToBoolean: ", err)

		return false
	}

	return boolValue
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// default configuration
	m.Html = htmlminify.Minifier{
		KeepConditionalComments: false,
		KeepSpecialComments:     false,
		KeepComments:            false,
		KeepWhitespace:          false,
		KeepDefaultAttrVals:     false,
		KeepDocumentTags:        false,
		KeepEndTags:             false,
		KeepQuotes:              false,
	}

	// TODO: improve mechanism to matching config with available minifier config
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}

		// block 0 to determine config for specific content type
		for d.NextBlock(0) {
			if d.NextArg() {
				return d.ArgErr()
			}

			// check content type name
			var configFor = d.Val()

			// block 1 to set config detail for specific content type
			for d.NextBlock(1) {
				if configFor == "html" {
					if d.Val() == "KeepComments" {
						// get value from config
						d.NextArg()
						m.Html.KeepComments = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepWhitespace" {
						// get value from config
						d.NextArg()
						m.Html.KeepWhitespace = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepDefaultAttrVals" {
						// get value from config
						d.NextArg()
						m.Html.KeepDefaultAttrVals = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepDocumentTags" {
						// get value from config
						d.NextArg()
						m.Html.KeepDocumentTags = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepEndTags" {
						// get value from config
						d.NextArg()
						m.Html.KeepEndTags = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepQuotes" {
						// get value from config
						d.NextArg()
						m.Html.KeepQuotes = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepConditionalComments" {
						// get value from config
						d.NextArg()
						m.Html.KeepConditionalComments = converterStrToBoolean(d.Val())
					}

					if d.Val() == "KeepSpecialComments" {
						// get value from config
						d.NextArg()
						m.Html.KeepSpecialComments = converterStrToBoolean(d.Val())
					}
				}
			}
		}
	}

	return nil
}
