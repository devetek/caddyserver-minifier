package minifier

import (
	"io"

	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

type responseMinifier struct {
	*caddyhttp.ResponseWriterWrapper
	handler *Middleware
}

func (fw *responseMinifier) WriteHeader(status int) {
	// we don't know the length after replacements since
	// we're not buffering it all to find out
	fw.Header().Del("Content-Length")

	fw.ResponseWriterWrapper.WriteHeader(status)
}

func (fw *responseMinifier) Write(d []byte) (int, error) {
	var writer io.WriteCloser
	var mediatype = fw.ResponseWriter.Header().Get("Content-Type")

	if mediatype == "text/html" {
		writer = fw.handler.minify.Writer(mediatype, fw.ResponseWriter)

		defer writer.Close()

		return writer.Write(d)
	}

	return fw.ResponseWriter.Write(d)
}
