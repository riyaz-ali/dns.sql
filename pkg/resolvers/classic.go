package resolvers

import (
	"crypto/tls"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"net"
	"net/url"
)

// ClassicResolver implements classic dns resolver for udp, tcp and tcp-tls protocols
type ClassicResolver struct {
	client *dns.Client
	server string
}

func NewClassicResolver(s string) (_ *ClassicResolver, err error) {
	var ep *url.URL
	if ep, err = url.Parse(s); err != nil {
		return nil, err
	}

	var scheme, server, port = ep.Scheme, ep.Hostname(), ep.Port()
	if scheme == "" || server == "" {
		return nil, errors.Errorf("invalid remote address: %s", s)
	} else if port == "" {
		return nil, errors.Errorf("specify remote port for %s", scheme)
	}

	var client = &dns.Client{Net: scheme}
	
	if query := ep.Query(); scheme == "tls" && query.Has("hostname") {
		client.Net = "tcp-tls"
		client.TLSConfig = &tls.Config{ServerName: query.Get("hostname")}
	} else if scheme == "tls" && !query.Has("hostname") {
		return nil, errors.Errorf("provide 'hostname' for use with TLS")
	}

	return &ClassicResolver{server: net.JoinHostPort(server, port), client: client}, nil
}

func (rslv *ClassicResolver) Lookup(ques dns.Question) (*dns.Msg, error) {
	var query = new(dns.Msg)
	query.Id, query.Question, query.RecursionDesired = dns.Id(), []dns.Question{ques}, true

	msg, _, err := rslv.client.Exchange(query, rslv.server)
	if err == nil && msg.Truncated {
		var n = rslv.client.Net
		rslv.client.Net = "tcp"

		defer func() { rslv.client.Net = n }() // restore after we're done
		return rslv.Lookup(ques)
	}

	return msg, err
}
