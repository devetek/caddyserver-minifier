package minifier

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
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

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
// Syntax:
//
//	minifier {
//		disable <html|css|js|svg>...
//		html {
//			<Option> <value>
//		}
//		css { ... }
//		js  { ... }
//		svg { ... }
//	}
//
// Options are the exported fields of the matching tdewolff/minify
// Minifier struct (e.g. KeepComments, Precision, KeepVarNames).
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume directive name
	if d.NextArg() {
		return d.ArgErr()
	}

	for d.NextBlock(0) {
		section := strings.ToLower(d.Val())

		if section == "disable" {
			args := d.RemainingArgs()
			if len(args) == 0 {
				return d.ArgErr()
			}
			for _, t := range args {
				if !isSupportedType(t) {
					return d.Errf("unsupported content type %q, expected one of %v", t, supportedTypes)
				}
			}
			m.Disable = append(m.Disable, args...)
			continue
		}

		var target any
		switch section {
		case typeHTML:
			target = &m.Html
		case typeCSS:
			target = &m.Css
		case typeJS:
			target = &m.Js
		case typeSVG:
			target = &m.Svg
		default:
			return d.Errf("unrecognized subdirective %q, expected disable or one of %v", d.Val(), supportedTypes)
		}
		if d.NextArg() {
			return d.ArgErr()
		}

		for d.NextBlock(1) {
			if err := setOption(d, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// setOption assigns the arguments on the current line to the exported
// field of target (a pointer to a minifier struct) named by d.Val().
func setOption(d *caddyfile.Dispenser, target any) error {
	name := d.Val()
	v := reflect.ValueOf(target).Elem()
	t := v.Type()

	var field reflect.Value
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() && strings.EqualFold(t.Field(i).Name, name) {
			field = v.Field(i)
			break
		}
	}
	if !field.IsValid() {
		return d.Errf("unknown option %q, expected one of %v", name, optionNames(t))
	}

	args := d.RemainingArgs()

	switch field.Kind() {
	case reflect.Bool:
		if len(args) != 1 {
			return d.ArgErr()
		}
		b, err := strconv.ParseBool(args[0])
		if err != nil {
			return d.Errf("option %s: invalid boolean %q", name, args[0])
		}
		field.SetBool(b)
	case reflect.Int:
		if len(args) != 1 {
			return d.ArgErr()
		}
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return d.Errf("option %s: invalid integer %q", name, args[0])
		}
		field.SetInt(int64(n))
	case reflect.Slice:
		if len(args) == 0 {
			return d.ArgErr()
		}
		field.Set(reflect.ValueOf(args))
	case reflect.Array:
		if len(args) != field.Len() {
			return d.Errf("option %s: expected %d values, got %d", name, field.Len(), len(args))
		}
		for i, a := range args {
			field.Index(i).SetString(a)
		}
	default:
		return d.Errf("option %s is not configurable from the Caddyfile", name)
	}

	return nil
}

func optionNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			names = append(names, t.Field(i).Name)
		}
	}
	return names
}
