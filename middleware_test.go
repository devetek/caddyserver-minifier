package minifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

const (
	sampleHTML = "<!DOCTYPE html>\n<html>\n  <head>  <title> Hello </title>  </head>\n  <body>\n    <!-- comment -->\n    <p   class=\"x\">  Hello   world  </p>\n  </body>\n</html>\n"
	sampleCSS  = "body {\n  color : #ff0000 ;\n  margin : 0px 0px 0px 0px ;\n}\n"
	sampleJS   = "function add ( first , second ) {\n  // comment\n  return first + second ;\n}\n"
	sampleSVG  = "<?xml version=\"1.0\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\"   width=\"10\" height=\"10\">\n  <!-- comment -->\n  <rect x=\"0\" y=\"0\" width=\"10\" height=\"10\" />\n</svg>\n"
)

func newMiddleware(t *testing.T, caddyfileInput string) *Middleware {
	t.Helper()
	m := new(Middleware)
	if caddyfileInput != "" {
		if err := m.UnmarshalCaddyfile(caddyfile.NewTestDispenser(caddyfileInput)); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	t.Cleanup(cancel)
	if err := m.Provision(ctx); err != nil {
		t.Fatalf("provision: %v", err)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	return m
}

// serve runs the middleware in front of next and returns the recorded response.
func serve(t *testing.T, m *Middleware, method string, next caddyhttp.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, "/", nil)
	if err := m.ServeHTTP(w, r, next); err != nil {
		t.Fatalf("serve: %v", err)
	}
	return w
}

func static(contentType, body string, extra ...string) caddyhttp.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		for i := 0; i+1 < len(extra); i += 2 {
			w.Header().Set(extra[i], extra[i+1])
		}
		// write in small chunks to make sure the body is minified as a whole
		for i := 0; i < len(body); i += 7 {
			end := min(i+7, len(body))
			if _, err := w.Write([]byte(body[i:end])); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestMinifiesEachType(t *testing.T) {
	m := newMiddleware(t, "")

	tests := []struct {
		name, contentType, body, want string
	}{
		{"html", "text/html; charset=utf-8", sampleHTML, `<!doctype html><title>Hello</title><p class=x>Hello world`},
		{"css", "text/css; charset=utf-8", sampleCSS, `body{color:red;margin:0}`},
		{"js application", "application/javascript", sampleJS, `function add(e,t){return e+t}`},
		{"js text", "text/javascript; charset=utf-8", sampleJS, `function add(e,t){return e+t}`},
		{"svg", "image/svg+xml", sampleSVG, `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect x="0" y="0" width="10" height="10"/></svg>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(t, m, http.MethodGet, static(tc.contentType, tc.body))
			got := w.Body.String()
			if got != tc.want {
				t.Fatalf("body mismatch\n got: %q\nwant: %q", got, tc.want)
			}
			if cl := w.Header().Get("Content-Length"); cl != strconv.Itoa(len(got)) {
				t.Fatalf("Content-Length = %s, want %d", cl, len(got))
			}
		})
	}
}

// file_server writes through io.Copy / ReadFrom, which previously bypassed
// the minifier entirely.
func TestMinifiesServeContent(t *testing.T) {
	m := newMiddleware(t, "")
	w := serve(t, m, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("ETag", `"abc"`)
		http.ServeContent(w, r, "style.css", time.Time{}, strings.NewReader(sampleCSS))
		return nil
	})
	if got := w.Body.String(); got != "body{color:red;margin:0}" {
		t.Fatalf("unexpected body %q", got)
	}
	if et := w.Header().Get("ETag"); et != `W/"abc"` {
		t.Fatalf("ETag = %q, want weak etag", et)
	}
	if w.Header().Get("Accept-Ranges") != "" {
		t.Fatalf("Accept-Ranges should be removed")
	}
}

func TestPassThrough(t *testing.T) {
	m := newMiddleware(t, "")

	tests := []struct {
		name   string
		method string
		next   caddyhttp.HandlerFunc
	}{
		{"unsupported type", http.MethodGet, static("application/json", `{ "a" : 1 }`)},
		{"compressed", http.MethodGet, static("text/css", sampleCSS, "Content-Encoding", "gzip")},
		{"no-transform", http.MethodGet, static("text/css", sampleCSS, "Cache-Control", "public, no-transform")},
		{"head", http.MethodHead, static("text/css", sampleCSS)},
		{"error status", http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "text/css")
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(sampleCSS))
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(t, m, tc.method, tc.next)
			ref := httptest.NewRecorder()
			_ = tc.next(ref, httptest.NewRequest(tc.method, "/", nil))
			if w.Body.String() != ref.Body.String() || w.Code != ref.Code {
				t.Fatalf("response was modified: %d %q", w.Code, w.Body.String())
			}
		})
	}
}

func TestDisable(t *testing.T) {
	m := newMiddleware(t, "minifier {\n disable css js\n}")
	if got := serve(t, m, http.MethodGet, static("text/css", sampleCSS)).Body.String(); got != sampleCSS {
		t.Fatalf("css should not be minified, got %q", got)
	}
	if got := serve(t, m, http.MethodGet, static("application/javascript", sampleJS)).Body.String(); got != sampleJS {
		t.Fatalf("js should not be minified, got %q", got)
	}
	if got := serve(t, m, http.MethodGet, static("image/svg+xml", sampleSVG)).Body.String(); got == sampleSVG {
		t.Fatalf("svg should still be minified")
	}
}

func TestOptions(t *testing.T) {
	m := newMiddleware(t, "minifier {\n html {\n  KeepComments true\n }\n js {\n  KeepVarNames true\n }\n}")
	if got := serve(t, m, http.MethodGet, static("text/html", sampleHTML)).Body.String(); !strings.Contains(got, "<!-- comment -->") {
		t.Fatalf("html comment should be kept, got %q", got)
	}
	if got := serve(t, m, http.MethodGet, static("text/javascript", sampleJS)).Body.String(); got != "function add(first,second){return first+second}" {
		t.Fatalf("js var names should be kept, got %q", got)
	}
}

func TestUnmarshalCaddyfile(t *testing.T) {
	m := new(Middleware)
	input := `minifier {
		disable svg
		html {
			KeepSpecialComments true
			KeepWhitespace true
			TemplateDelims {{ }}
		}
		css {
			Precision 3
			Inline true
		}
		js {
			KeepVarNames true
			Version 2020
		}
		svg {
			KeepComments true
			KeepNamespaces inkscape sodipodi
		}
	}`
	if err := m.UnmarshalCaddyfile(caddyfile.NewTestDispenser(input)); err != nil {
		t.Fatal(err)
	}
	switch {
	case !m.Html.KeepSpecialComments || !m.Html.KeepWhitespace || m.Html.TemplateDelims != [2]string{"{{", "}}"}:
		t.Fatalf("html: %+v", m.Html)
	case m.Css.Precision != 3 || !m.Css.Inline:
		t.Fatalf("css: %+v", m.Css)
	case !m.Js.KeepVarNames || m.Js.Version != 2020:
		t.Fatalf("js: %+v", m.Js)
	case !m.Svg.KeepComments || len(m.Svg.KeepNamespaces) != 2:
		t.Fatalf("svg: %+v", m.Svg)
	case len(m.Disable) != 1 || m.Disable[0] != "svg":
		t.Fatalf("disable: %v", m.Disable)
	}
}

func TestUnmarshalCaddyfileErrors(t *testing.T) {
	for name, input := range map[string]string{
		"directive arg":   "minifier foo",
		"unknown section": "minifier {\n xml\n}",
		"unknown option":  "minifier {\n html {\n  KeepNothing true\n }\n}",
		"bad bool":        "minifier {\n html {\n  KeepComments yes-please\n }\n}",
		"bad int":         "minifier {\n css {\n  Precision high\n }\n}",
		"missing value":   "minifier {\n html {\n  KeepComments\n }\n}",
		"bad disable":     "minifier {\n disable xml\n}",
		"empty disable":   "minifier {\n disable\n}",
	} {
		t.Run(name, func(t *testing.T) {
			if err := new(Middleware).UnmarshalCaddyfile(caddyfile.NewTestDispenser(input)); err == nil {
				t.Fatalf("expected error for %q", input)
			}
		})
	}
}

func TestValidateRejectsUnknownDisable(t *testing.T) {
	m := &Middleware{Disable: []string{"xml"}}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
