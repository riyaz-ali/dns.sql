package resolvers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
)

// HttpResolver implements dns-over-https resolver.
type HttpResolver struct {
	client *http.Client
	server string
}

func NewHttpResolver(s string) (_ *HttpResolver, err error) {
	var ep *url.URL
	if ep, err = url.Parse(s); err != nil {
		return nil, errors.Wrapf(err, "%s is not a valid https endpoint", s)
	}

	if ep.Scheme != "https" {
		return nil, fmt.Errorf("missing https in %s", s)
	}

	return &HttpResolver{client: &http.Client{}, server: ep.String()}, nil
}

func (rslv *HttpResolver) Lookup(ques dns.Question) (_ *dns.Msg, err error) {
	var query = new(dns.Msg)
	query.Id, query.Question, query.RecursionDesired = dns.Id(), []dns.Question{ques}, true

	var buf []byte
	if buf, err = query.Pack(); err != nil {
		return nil, err
	}

	var resp *http.Response
	if resp, err = rslv.client.Post(rslv.server, "application/dns-message", bytes.NewReader(buf)); err != nil {
		return nil, err
	}

	// if POST isn't allowed, try GET
	if resp.StatusCode == http.StatusMethodNotAllowed {
		var u *url.URL
		if u, err = url.Parse(rslv.server); err != nil {
			return nil, err
		}

		u.RawQuery = fmt.Sprintf("dns=%v", base64.RawURLEncoding.EncodeToString(buf))
		if resp, err = rslv.client.Get(u.String()); err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from nameserver %s", resp.Status)
	}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	if err = query.Unpack(body); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack")
	}

	return query, nil
}
