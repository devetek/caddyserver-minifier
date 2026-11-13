## Caddyserver v2 Plugin - Minifier

Caddyserver v2 plugin that minifies HTML, CSS, JavaScript and SVG responses on the fly. It uses [tdewolff's library](https://github.com/tdewolff/minify) so, let's thank him!.

Tested with Caddy **v2.11.9** (requires Go 1.26+ to build).

## Build

```sh
xcaddy build v2.11.9 --with github.com/devetek/caddyserver-minifier
```

## How it works

Responses with one of the following `Content-Type` are buffered, minified in a single pass and sent with an updated `Content-Length`:

| Type | Content-Type                                                          |
| ---- | --------------------------------------------------------------------- |
| html | `text/html`                                                           |
| css  | `text/css`                                                            |
| js   | `application/javascript`, `text/javascript`, `application/x-javascript`, ... |
| svg  | `image/svg+xml`                                                       |

Inline `<style>`, `<script>` and `<svg>` inside HTML are minified as well, using the CSS/JS/SVG configuration.

Responses are passed through untouched when they are:

- not `2xx`, or are `204`/`206` responses, or a `HEAD` request
- already compressed (`Content-Encoding` is set, e.g. a gzipped upstream behind `reverse_proxy`)
- marked with `Cache-Control: no-transform`

If minification fails the original body is served and a warning is logged. A strong `ETag` is converted to a weak one since the minified body is no longer byte-identical.

## Syntax

Because this directive does not come standard with Caddy, you may use `route` to order it the way you want. Place `encode` before `minifier` so the response is minified before it is compressed. For example:

Minimum configuration (all types enabled with default settings):
```sh
http://localhost:9200 {
	route {
		encode zstd gzip
		minifier
		reverse_proxy localhost:8097
	}
}
```

Disable some content types:
```sh
http://localhost:9200 {
	route {
		minifier {
			disable js svg
		}
		file_server
	}
}
```

Full configuration:
```sh
http://localhost:9200 {
	route {
		minifier {
			html {
				KeepSpecialComments true
				KeepConditionalComments true
				KeepComments true
				KeepWhitespace true
				KeepDefaultAttrVals true
				KeepDocumentTags true
				KeepEndTags true
				KeepQuotes true
				TemplateDelims {{ }}
			}
			css {
				Precision 0
				Inline false
				Version 0
			}
			js {
				Precision 0
				KeepVarNames true
				Version 0
			}
			svg {
				KeepComments true
				Precision 0
				KeepNamespaces inkscape sodipodi
			}
		}
		reverse_proxy localhost:8097
	}
}
```

Options map 1:1 to the exported fields of the tdewolff/minify `Minifier` structs, see [html](https://pkg.go.dev/github.com/tdewolff/minify/v2/html#Minifier), [css](https://pkg.go.dev/github.com/tdewolff/minify/v2/css#Minifier), [js](https://pkg.go.dev/github.com/tdewolff/minify/v2/js#Minifier) and [svg](https://pkg.go.dev/github.com/tdewolff/minify/v2/svg#Minifier). Unknown options or invalid values are rejected when the config is loaded.

JSON configuration uses the same names:
```json
{
  "handler": "minifier",
  "disable": ["svg"],
  "Html": { "KeepComments": true },
  "Css": { "Precision": 3 },
  "Js": { "KeepVarNames": true },
  "Svg": { "KeepComments": true }
}
```

## Credit

Special thanks to @mholt for making a good documentation on how to use unstandard Caddy plugins, [mholt/caddy-webdav](https://github.com/mholt/caddy-webdav)
