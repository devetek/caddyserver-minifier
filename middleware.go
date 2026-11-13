package minifier

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tdewolff/minify/v2"
	"go.uber.org/zap"

	cssminify "github.com/tdewolff/minify/v2/css"
	htmlminify "github.com/tdewolff/minify/v2/html"
	jsminify "github.com/tdewolff/minify/v2/js"
	svgminify "github.com/tdewolff/minify/v2/svg"
)

// Supported content type groups. They can be referenced by name in the
// `disable` option.
const (
	typeHTML = "html"
	typeCSS  = "css"
	typeJS   = "js"
	typeSVG  = "svg"
)

var supportedTypes = []string{typeHTML, typeCSS, typeJS, typeSVG}

// jsMimetypes matches every media type commonly used to serve JavaScript,
// plus "module" which is used for <script type="module"> inside HTML.
var jsMimetypes = regexp.MustCompile(`^(application|text)/(x-)?(java|ecma|j|live)script(1\.[0-5])?$|^module$`)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// Middleware minifies HTML, CSS, JavaScript and SVG responses on the fly.
type Middleware struct {
	// Html configures the HTML minifier.
	Html htmlminify.Minifier
	// Css configures the CSS minifier.
	Css cssminify.Minifier
	// Js configures the JavaScript minifier.
	Js jsminify.Minifier
	// Svg configures the SVG minifier.
	Svg svgminify.Minifier
	// Disable lists content types that must not be minified.
	// Valid values: html, css, js, svg.
	Disable []string `json:"disable,omitempty"`

	minify *minify.M
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.minifier",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *Middleware) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger()

	disabled := make(map[string]bool, len(m.Disable))
	for _, t := range m.Disable {
		disabled[strings.ToLower(t)] = true
	}

	m.minify = minify.New()
	if !disabled[typeHTML] {
		m.minify.Add("text/html", &m.Html)
	}
	if !disabled[typeCSS] {
		m.minify.Add("text/css", &m.Css)
	}
	if !disabled[typeJS] {
		m.minify.AddRegexp(jsMimetypes, &m.Js)
	}
	if !disabled[typeSVG] {
		m.minify.Add("image/svg+xml", &m.Svg)
	}

	return nil
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
	for _, t := range m.Disable {
		if !isSupportedType(t) {
			return fmt.Errorf("unsupported content type %q in disable, expected one of %v", t, supportedTypes)
		}
	}
	return nil
}

func isSupportedType(t string) bool {
	for _, s := range supportedTypes {
		if strings.EqualFold(t, s) {
			return true
		}
	}
	return false
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
//
// Responses with a minifiable Content-Type are buffered in full, minified
// in one pass and then written with a correct Content-Length. Everything
// else is streamed through untouched.
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.Method == http.MethodHead {
		return next.ServeHTTP(w, r)
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	rec := caddyhttp.NewResponseRecorder(w, buf, m.shouldBuffer)
	if err := next.ServeHTTP(rec, r); err != nil {
		return err
	}
	if !rec.Buffered() || rec.Status() == 0 {
		return rec.WriteResponse()
	}

	out := bufPool.Get().(*bytes.Buffer)
	out.Reset()
	defer bufPool.Put(out)

	body := buf
	if err := m.minify.Minify(rec.Header().Get("Content-Type"), out, bytes.NewReader(buf.Bytes())); err != nil {
		// never break a response because of a minification problem:
		// serve the original body instead
		m.logger.Warn("minification failed, serving original response",
			zap.String("uri", r.RequestURI),
			zap.String("content_type", rec.Header().Get("Content-Type")),
			zap.Error(err))
	} else {
		body = out
	}

	header := w.Header()
	header.Set("Content-Length", strconv.Itoa(body.Len()))
	// byte ranges of the original representation no longer apply
	header.Del("Accept-Ranges")
	// the minified body is semantically equivalent, not byte-identical
	if etag := header.Get("ETag"); etag != "" && !strings.HasPrefix(etag, "W/") {
		header.Set("ETag", "W/"+etag)
	}

	w.WriteHeader(rec.Status())
	_, err := w.Write(body.Bytes())
	return err
}

// shouldBuffer decides, once the response headers are known, whether the
// response is a candidate for minification.
func (m *Middleware) shouldBuffer(status int, header http.Header) bool {
	if status < 200 || status >= 300 || status == http.StatusNoContent || status == http.StatusPartialContent {
		return false
	}
	// already compressed (e.g. by an upstream behind reverse_proxy)
	if ce := header.Get("Content-Encoding"); ce != "" && !strings.EqualFold(ce, "identity") {
		return false
	}
	// RFC 9111: intermediaries must not transform the payload
	if strings.Contains(strings.ToLower(header.Get("Cache-Control")), "no-transform") {
		return false
	}
	ct := header.Get("Content-Type")
	if ct == "" {
		return false
	}
	_, _, fn := m.minify.Match(ct)
	return fn != nil
}
