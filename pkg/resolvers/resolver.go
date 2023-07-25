package resolvers

import (
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"net/url"
)

// Resolver represents an entity that can query a nameserver (or a recursive resolver), and return
// results for a given dns query.
type Resolver interface {
	// Lookup performs a dns lookup and returns a result for the given dns.Question
	Lookup(dns.Question) (*dns.Msg, error)
}

// NewResolver returns an instance of Resolver based on the given server address.
func NewResolver(server string) (_ Resolver, err error) {
	var remote *url.URL

	// parse server into a remote address
	if remote, err = url.Parse(server); err != nil {
		return nil, err
	}

	switch sch := remote.Scheme; sch {
	case "udp", "tcp", "tls":
		return NewClassicResolver(remote.String())

	case "https":
		return NewHttpResolver(remote.String())

	default:
		return nil, errors.Errorf("no registered resolver for %q protocol", sch)
	}
}
