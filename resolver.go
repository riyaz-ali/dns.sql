package dns

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	rslvr "go.riyazali.net/dns.sql/pkg/resolvers"
	"go.riyazali.net/sqlite"
)

const (
	ColumnDomain     = iota // name of the resource / domain to query for
	ColumnSection           // the response section where this resource record appears
	ColumnClass             // the DNS class to query for (defaults to IN)
	ColumnType              // the DNS Record Resource type (defaults to A)
	ColumnTTL               // TTL in seconds, indicating how long this resource record can be cached
	ColumnNameserver        // the nameserver to contact or the one that responded to the query
	ColumnData              // JSON encoded resource data specific to the type
)

const (
	ColumnPartial = iota // user-provided partial domain
	ColumnNdots          // user-provided ndots value
	ColumnFqdn           // resolved FQDN from search-list
)

// ResolverModule implements sqlite.Module for the dns() table-valued function.
type ResolverModule struct{}

func (r *ResolverModule) Connect(c *sqlite.Conn, _ []string, declare func(string) error) (sqlite.VirtualTable, error) {
	const query = "CREATE TABLE dns (domain, section, class, type, ttl, nameserver, data)"
	if err := declare(query); err != nil {
		return nil, err
	}

	return &ResolverTable{}, nil
}

// ResolverTable implements sqlite.VirtualTable for the dns() table-valued function.
// It receives the user query and is responsible to execute the corresponding dns query and return its result.
type ResolverTable struct{}

func (r *ResolverTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// argv and bitmap together defines the contract between BestIndex() and Filter()
	// each byte in the bitmap contains relevant information about the corresponding value in Filter()
	// most importantly, it describes the column index for that constraint value
	var argv = 1
	var bitmap []byte

	var out = &sqlite.IndexInfoOutput{
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(input.Constraints)),
	}

	// flag to ensure domain name and nameserver are constrained
	var required = byte(0)

	for i, cons := range input.Constraints {
		switch col, op := cons.ColumnIndex, cons.Op; col {
		case ColumnDomain:
			{
				if op != sqlite.INDEX_CONSTRAINT_EQ {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to operation is supported")
				}

				if op == sqlite.INDEX_CONSTRAINT_EQ && cons.Usable {
					out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv, Omit: true}
					bitmap, argv = append(bitmap, byte(col)), argv+1
					required = required | 0b01
				}
			}
		case ColumnNameserver, ColumnType, ColumnClass:
			{
				if op != sqlite.INDEX_CONSTRAINT_EQ {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to operation is supported")
				}

				if op == sqlite.INDEX_CONSTRAINT_EQ && cons.Usable {
					out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: argv} // need omit=false here to maintain invariance of equals-to operation
					bitmap, argv = append(bitmap, byte(col)), argv+1

					if col == ColumnNameserver {
						required = required | 0b10
					}
				}
			}
		}
	}

	if required != 0b11 { // value signifies that all (currently 2) required constraints are met
		return out, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "required constraints not met")
	}

	// pass the bitmap as string to Filter() routine
	out.IndexString = base64.StdEncoding.EncodeToString(bitmap)

	return out, nil
}

func (r *ResolverTable) Open() (sqlite.VirtualCursor, error) { return &ResolverCursor{}, nil }
func (r *ResolverTable) Disconnect() error                   { return nil }
func (r *ResolverTable) Destroy() error                      { return nil }

type rr struct {
	section string
	record  dns.RR
}

// ResolverCursor implements sqlite.VirtualCursor for the dns() table-valued function.
// It represents an open cursor in the returned result-set for the user query.
type ResolverCursor struct {
	ns      string
	pos     int
	records []*rr
}

func (cur *ResolverCursor) Filter(_ int, str string, values ...sqlite.Value) (err error) {
	var server string
	var domain string
	var class, typ = "IN", "A"

	var bitmap, _ = base64.StdEncoding.DecodeString(str)
	for n, val := range values {
		var col = bitmap[n]
		switch {
		case col == ColumnDomain:
			domain = val.Text()

		case col == ColumnType:
			typ = val.Text()

		case col == ColumnClass:
			class = val.Text()

		case col == ColumnNameserver:
			server = val.Text()
		}
	}

	var rslv rslvr.Resolver
	if rslv, err = rslvr.NewResolver(server); err != nil {
		return err
	}

	var ques = dns.Question{Name: domain, Qtype: dns.StringToType[typ], Qclass: dns.StringToClass[class]}

	var msg *dns.Msg
	if msg, err = rslv.Lookup(ques); err != nil {
		return err
	}

	cur.ns, cur.pos = server, 0
	cur.records = make([]*rr, 0, len(msg.Answer)+len(msg.Ns)+len(msg.Extra))

	for _, record := range msg.Answer {
		cur.records = append(cur.records, &rr{section: "answer", record: record})
	}

	for _, record := range msg.Ns {
		cur.records = append(cur.records, &rr{section: "authority", record: record})
	}

	for _, record := range msg.Extra {
		cur.records = append(cur.records, &rr{section: "extra", record: record})
	}

	return nil
}

func (cur *ResolverCursor) Column(ctx *sqlite.VirtualTableContext, col int) error {
	var rr = cur.records[cur.pos]

	switch col {
	case ColumnDomain:
		ctx.ResultText(rr.record.Header().Name)
	case ColumnSection:
		ctx.ResultText(rr.section)
	case ColumnClass:
		ctx.ResultText(dns.ClassToString[rr.record.Header().Class])
	case ColumnType:
		ctx.ResultText(dns.TypeToString[rr.record.Header().Rrtype])
	case ColumnTTL:
		ctx.ResultInt(int(rr.record.Header().Ttl))
	case ColumnNameserver:
		ctx.ResultText(cur.ns)
	case ColumnData:
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(rr.record); err != nil {
			return err
		}
		ctx.ResultSubType(74 /* subtype for JSON */)
		ctx.ResultBlob(buf.Bytes())
	}

	return nil
}

func (cur *ResolverCursor) Next() error           { cur.pos++; return nil }
func (cur *ResolverCursor) Eof() bool             { return cur.pos >= len(cur.records) }
func (cur *ResolverCursor) Rowid() (int64, error) { return int64(cur.pos), nil }
func (cur *ResolverCursor) Close() error          { return nil }

// SearchList implements sqlite.VirtualTable for SearchList() table-valued function
//
// The SearchList() appends to the given domain, options from the search-list field specified
// in the system's dns config.
type SearchList struct{}

func (s *SearchList) Connect(_ *sqlite.Conn, _ []string, declare func(string) error) (_ sqlite.VirtualTable, err error) {
	if err = declare("CREATE TABLE search (partial HIDDEN, ndots HIDDEN, fqdn TEXT PRIMARY KEY) WITHOUT ROWID"); err != nil {
		return nil, err
	}

	return &SearchListTable{}, nil
}

// SearchListTable implements sqlite.VirtualTable for SearchList() table-valued function
type SearchListTable struct{}

func (tab *SearchListTable) BestIndex(input *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	var out = &sqlite.IndexInfoOutput{
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(input.Constraints)),
	}

	var domainConstrained = false

	for i, cons := range input.Constraints {
		switch col, op := cons.ColumnIndex, cons.Op; col {
		//case ColumnFqdn:
		//	return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "no constraint supported on fqdn")

		case ColumnPartial, ColumnNdots:
			{
				if op != sqlite.INDEX_CONSTRAINT_EQ {
					return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "only equals-to operation is supported")
				}

				if op == sqlite.INDEX_CONSTRAINT_EQ && cons.Usable {
					out.ConstraintUsage[i] = &sqlite.ConstraintUsage{ArgvIndex: col + 1}
				}

				if col == ColumnPartial {
					domainConstrained = true
				}
			}
		}
	}

	if !domainConstrained {
		return nil, sqlite.Error(sqlite.SQLITE_CONSTRAINT, "partial domain value must be provided")
	}

	return out, nil
}

func (tab *SearchListTable) Open() (sqlite.VirtualCursor, error) { return &SearchListCursor{}, nil }
func (tab *SearchListTable) Disconnect() error                   { return nil }
func (tab *SearchListTable) Destroy() error                      { return nil }

// SearchListCursor implements sqlite.VirtualCursor for SearchList() table-valued function
type SearchListCursor struct {
	pos     int
	partial string
	ndots   int
	fqdn    []string
}

func (cur *SearchListCursor) Filter(_ int, _ string, values ...sqlite.Value) (err error) {
	partial, ndots := values[0].Text(), 1
	if len(values) > 1 {
		ndots = values[1].Int()
	}
	cur.pos, cur.partial, cur.ndots = 0, partial, ndots

	// if this domain is already fully qualified, no append needed.
	if dns.IsFqdn(partial) {
		cur.fqdn = []string{partial}
		return nil
	}

	// fetch search list from system configuration
	var config *dns.ClientConfig
	if config, err = dns.ClientConfigFromFile("/etc/resolv.conf"); err != nil {
		return errors.Wrapf(err, "failed to read system configuration")
	}

	var searchList = config.Search
	var fq = dns.Fqdn(partial)

	// if name has enough dots, try that first
	if dns.CountLabel(partial) > ndots {
		cur.fqdn = append(cur.fqdn, fq)
	}

	for _, s := range searchList {
		cur.fqdn = append(cur.fqdn, dns.Fqdn(fq+s))
	}

	if dns.CountLabel(partial) <= ndots {
		cur.fqdn = append(cur.fqdn, fq)
	}

	return nil
}

func (cur *SearchListCursor) Column(c *sqlite.VirtualTableContext, col int) error {
	switch col {
	case ColumnPartial:
		c.ResultText(cur.partial)
	case ColumnNdots:
		c.ResultInt(cur.ndots)
	case ColumnFqdn:
		c.ResultText(cur.fqdn[cur.pos])
	}

	return nil
}

func (cur *SearchListCursor) Next() error           { cur.pos++; return nil }
func (cur *SearchListCursor) Eof() bool             { return cur.pos >= len(cur.fqdn) }
func (cur *SearchListCursor) Rowid() (int64, error) { return 0, nil }
func (cur *SearchListCursor) Close() error          { return nil }
