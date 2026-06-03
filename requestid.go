package requestid

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	nanoid "github.com/matoous/go-nanoid/v2"
)

const defaultLength = 21

const placeholder = "http.request_id"

func init() { //nolint:gochecknoinits // caddy plugin registration in the init func
	caddy.RegisterModule(&RequestID{})
	httpcaddyfile.RegisterHandlerDirective("request_id", parseCaddyfile)
}

type RequestID struct {
	Length     int            `json:"length"`
	Additional map[string]int `json:"additional,omitempty"`
}

func (*RequestID) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.request_id",
		New: func() caddy.Module { return new(RequestID) },
	}
}

func (m *RequestID) Provision(_ caddy.Context) error {
	if m.Length == 0 {
		m.Length = defaultLength
	} else if m.Length < 0 {
		return fmt.Errorf("length cannot be negative, got %d", m.Length)
	}
	for name, length := range m.Additional {
		if length < 1 {
			return fmt.Errorf("additional ID %q: length must be at least 1, got %d", name, length)
		}
	}
	return nil
}

func (m *RequestID) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl, ok := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	if !ok {
		return next.ServeHTTP(w, r)
	}
	if err := setID(repl, placeholder, m.Length); err != nil {
		return err
	}
	for name, length := range m.Additional {
		if err := setID(repl, placeholder+"."+name, length); err != nil {
			return err
		}
	}
	return next.ServeHTTP(w, r)
}

func setID(repl *caddy.Replacer, key string, length int) error {
	id, err := nanoid.New(length)
	if err != nil {
		return fmt.Errorf("generating ID for %q: %w", key, err)
	}
	repl.Set(key, id)
	return nil
}

func (m *RequestID) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()
	if d.NextArg() {
		length, err := strconv.Atoi(d.Val())
		if err != nil {
			return d.Errf("invalid length %q: %v", d.Val(), err)
		}
		m.Length = length
	}
	if d.NextArg() {
		return d.ArgErr()
	}
	for d.NextBlock(0) {
		name := d.Val()
		if !d.NextArg() {
			return d.ArgErr()
		}
		length, err := strconv.Atoi(d.Val())
		if err != nil {
			return d.Errf("invalid length %q for %q: %v", d.Val(), name, err)
		}
		if _, ok := m.Additional[name]; ok {
			return d.Errf("duplicate key: %s", name)
		}
		if m.Additional == nil {
			m.Additional = make(map[string]int)
		}
		m.Additional[name] = length
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	m := new(RequestID)
	if err := m.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return m, nil
}

var (
	_ caddy.Provisioner           = (*RequestID)(nil)
	_ caddyhttp.MiddlewareHandler = (*RequestID)(nil)
	_ caddyfile.Unmarshaler       = (*RequestID)(nil)
)
