package dns

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"time"
)

// FQDN implements sqlite.ScalarFunction for FQDN() SQL function
type FQDN struct{}

func (_ *FQDN) Deterministic() bool { return true }
func (_ *FQDN) Args() int           { return 1 }
func (_ *FQDN) Apply(c *sqlite.Context, values ...sqlite.Value) {
	c.ResultText(dns.Fqdn(values[0].Text()))
}

// ClassicResolver implements sqlite.ScalarFunction for ClassicResolver() SQL function.
//
// The ClassicResolver() function takes in remote address, protocol and other information
// about the resolver, and constructs a well-formed URL that can then be passed to 'nameserver' field of DNS() table-valued function.
type ClassicResolver struct{}

func (_ *ClassicResolver) Deterministic() bool { return true }
func (_ *ClassicResolver) Args() int           { return 3 }

func (_ *ClassicResolver) Apply(c *sqlite.Context, values ...sqlite.Value) {
	protocol, remote, port := values[0].Text(), values[1].Text(), values[2].Int()
	c.ResultText(fmt.Sprintf("%s://%s:%d", protocol, remote, port))
}

// TlsResolver implements sqlite.ScalarFunction for TlsResolver() SQL function.
//
// The TlsResolver() function takes in remote address, and other information, such as TLS Hostname,
// about the resolver, and constructs a well-formed URL that can then be passed to 'nameserver' field of DNS() table-valued function.
type TlsResolver struct{}

func (_ *TlsResolver) Deterministic() bool { return true }
func (_ *TlsResolver) Args() int           { return 3 }

func (_ *TlsResolver) Apply(c *sqlite.Context, values ...sqlite.Value) {
	remote, port, hostname := values[0].Text(), values[1].Int(), values[2].Text()
	c.ResultText(fmt.Sprintf("tls://%s:%d?hostname=%s", remote, port, hostname))
}

// SystemResolver implements sqlite.ScalarFunction for SystemResolver() SQL function.
//
// The SystemResolver() function returns a well-formed url describing the system's configured dns resolver.
type SystemResolver struct{}

func (_ *SystemResolver) Deterministic() bool { return false } // depends on external system values
func (_ *SystemResolver) Args() int           { return 0 }

func (_ *SystemResolver) Apply(c *sqlite.Context, _ ...sqlite.Value) {
	var err error

	var config *dns.ClientConfig
	if config, err = dns.ClientConfigFromFile("/etc/resolv.conf"); err != nil {
		c.ResultError(err)
		return
	}

	// choose a random server from the list
	rand.Seed(time.Now().Unix())
	var srv = config.Servers[rand.Intn(len(config.Servers))]

	var ep *url.URL
	if ip := net.ParseIP(srv); ip != nil {
		ep = new(url.URL)
		ep.Host = ip.String()
	} else {
		if ep, err = url.Parse(srv); err != nil {
			c.ResultError(errors.Wrapf(err, "failed to load default system resolver"))
			return
		}
	}

	if ep.Scheme == "" {
		ep.Scheme = "udp" // default to udp
	}

	if ep.Port() == "" {
		ep.Host = net.JoinHostPort(ep.Hostname(), "53")
	}

	var query = ep.Query()
	query.Set("ndots", strconv.Itoa(config.Ndots))
	for _, s := range config.Search {
		query.Add("search", s)
	}
	ep.RawQuery = query.Encode()

	c.ResultText(ep.String())
}
