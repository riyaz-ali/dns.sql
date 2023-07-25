🔎 dns.sql
----------

Have you ever wanted to query DNS using SQL? No? Well now you can 🤪

**`dns.sql`** is a `sqlite3` extension that allows you to query DNS using SQL. Load it into your `sqlite3` environment and run,

```sql
SELECT * FROM dns WHERE domain = FQDN('riyazali.net') AND nameserver = SystemResolver();
```

The above query yields a result similar to:

```
domain         section  class  type  ttl  nameserver                    data                                                        
-------------  -------  -----  ----  ---  --------------------  ------------------------
riyazali.net.  answer   IN     A     289  udp://192.168.1.1:53  {"A":"185.199.110.153"}                      

riyazali.net.  answer   IN     A     289  udp://192.168.1.1:53  {"A":"185.199.111.153"}                      

riyazali.net.  answer   IN     A     289  udp://192.168.1.1:53  {"A":"185.199.108.153"}                      

riyazali.net.  answer   IN     A     289  udp://192.168.1.1:53  {"A":"185.199.109.153"}
```

## Usage

To build as a `sqlite3` shared extension, you can use the `cmd/shared/shared.go` target.

```shell
> go build -o libdns.so -buildmode=c-shared cmd/shared/shared.go
```

Then, to load the extension into your `sqlite3` shell:

```
sqlite> .load libdns.so
```

## Schema

The module provides the following _virtual tables_ and _SQL functions_:

- **`DNS()`** is a _table-valued_ function module that provides the main lookup functionality. It contains the following columns:

  - `domain` is the [`FQDN`](https://en.wikipedia.org/wiki/Fully_qualified_domain_name) of the query / name being resolved.  
    It needs to be a fully-qualified name. Use `FQDN()` function  (defined below) to ensure the name you pass here is fully qualified.
  - `section` is the setion in the DNS response where this resource record appeared. Valid values include `answer`, `authority` and `extra`
  - `class` is the [class code](https://en.wikipedia.org/wiki/Domain_Name_System#Resource_records) of the DNS record
  - `type` is the [type of the resource records](https://en.wikipedia.org/wiki/List_of_DNS_record_types)
  - `ttl` is the [Time-to-live](https://en.wikipedia.org/wiki/Time_to_live) value before the record must be refetched / refreshed.
  - `nameserver` is either the authoritative or recursive nameserver that answered the query.  
    When querying, this is a required parameter and must be provided. Use one of `ClassicResolver()`, `TlsResolver()`
    or a formatted `http` url (for [`DoH`](https://en.wikipedia.org/wiki/DNS_over_HTTPS))
  - `data` is the JSON-formatted implementations of [`dns.RR`](https://pkg.go.dev/github.com/miekg/dns#RR)
 
- **`SearchList()`** is a _table-valued_ function module that provides search list resolution functionality. It contains the following columns:

  - `partial` is the user-provided partial input to the function. This is a `HIDDEN` column.
  - `ndots` is the user-provided value for the [`ndots` option](https://linux.die.net/man/5/resolv.conf). This is a `HIDDEN` column.
  - `fqdn` is the resolved FQDN based on system's search list and `ndots`

  Assuming system's search list is `ns1.svc.cluster.local`, `svc.cluster.local`, `cluster.local`, and `ndots` is `5` <sub>(example taken from http://redd.it/duj86x)</sub>
  ```
  SELECT * FROM search_list('app.ns2', 5)
  ```
  ```
  fqdn                          
  ------------------------------
  app.ns2.ns1.svc.cluster.local.
  app.ns2.svc.cluster.local.    
  app.ns2.cluster.local.        
  app.ns2. 
  ```

- **`FQDN(name)`** is a custom scalar function that takes in `name` and returns a formatted, fully-qualified domain name.

- **`ClassicResolver(protocol, host, port)`** is a custom scalar function and builds a well-formatted resolver url for use as a `dns.nameserver` constraint.  
  Supported protocol values include `udp` and `tcp`. Specify `53` as default port.

- **`TlsResolver(remote, port, hostname)`** is a custom scalar function and builds a well-formatted resolver url for use as a `dns.nameserver` constraint.
  It builds a url used by [`DoT`](https://en.wikipedia.org/wiki/DNS_over_TLS) resolver. `hostname` is used to verify the server's TLS certificate.

- **`SystemResolver()`** is a custom scalar function and builds a well-formatted resolver url for use as a `dns.nameserver` constraint.
  It reads from system's DNS configuration and returns a well-formed url. It reads from `/etc/resolv.conf`. This resolver is not supported on Windows.


To use [`DoH`](https://en.wikipedia.org/wiki/DNS_over_HTTPS), specify a valid url to a `DoH` service (like `https://cloudflare-dns.com/dns-query`), eg:

```sql
SELECT * FROM dns
  WHERE domain = FQDN('riyazali.net') AND nameserver = 'https://cloudflare-dns.com/dns-query';
```

## License

MIT License Copyright (c) 2023 Riyaz Ali. Refer to [LICENSE](./LICENSE) for full text.
