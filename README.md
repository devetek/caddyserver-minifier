## Caddyserver v2 Plugin - Minifier

Caddyserver v2 plugin that implements minification on-the-fly for CSS, HTML, JSON, SVG and XML. It uses [tdewolff's library](https://github.com/tdewolff/minify) so, let's thank him!.

## Syntax

Because this directive does not come standard with Caddy, you may use route to order it the way you want. For example:

```sh
http://localhost:9200 {
	route {
		minifier
		encode zstd gzip
		reverse_proxy localhost:8097
	}
}
```

## Todo

[ ] Filter supported `Content-Type`

```sh
minifier {
    support "text/html" "image/svg+xml"
}
```

## Credit

Special thanks to @mholt for making a good documentation on how to use unstandard Caddy plugins, [mholt/caddy-webdav](https://github.com/mholt/caddy-webdav)
