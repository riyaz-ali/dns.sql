# Recipes 🥘

Some recipes to keep handy!

## Implement `dig +trace` style tracer

This script is inspired by [this superuser post](https://superuser.com/a/715656). Enjoy 🙌

```sql
WITH RECURSIVE nameservers AS (
    -- First, find the addresses of the root resolvers and pick one
    SELECT 1 AS level, dns.* FROM dns
        WHERE rowid = 1 AND
              (domain = '.' AND type = 'NS' AND nameserver = SystemResolver())
    UNION ALL
    -- For each subsequent level, ask the resolver discovered in the last step
    -- to provide the A record (default) for the domain. If it isn't the
    -- authoritative nameserver, it'll respond with an NS record in the `authority` section.
    SELECT nameservers.level + 1, dns.* FROM dns, nameservers
        WHERE dns.rowid = 1 AND
              (dns.domain = FQDN(:domain) AND dns.section = 'authority' AND
                  dns.nameserver = ClassicResolver('udp', nameservers.data ->> 'Ns', 53))
    -- by the end we'll have a path (of resolvers) from root till the authoritative server
    -- (or an intermediary if the domain doesn't exist)
), records AS (
    -- for each level of resolver, repeat the question
    -- but this time record all the answers (ie. no rowid = 1 filter)
    SELECT ns.level, dns.* FROM dns, nameservers ns
        WHERE dns.domain = ns.domain AND dns.type = 'NS' AND dns.nameserver = ns.nameserver
)
SELECT domain, type, class, ttl, data ->> 'Ns' AS address, nameserver FROM records
UNION ALL
-- In the last step, we perform the actual address resolution of the domain we're tracing
-- by asking (what we think is) the authoritative nameserver for the domain
SELECT dns.domain, dns.type, dns.class, dns.ttl, dns.data ->> 'A' AS address, dns.nameserver
FROM dns, (SELECT *, MAX(level) FROM nameservers) ns
  WHERE dns.domain = FQDN(:domain) AND dns.nameserver = ClassicResolver('udp', ns.data ->> 'Ns', 53);
```

where `:domain` is the domain name you want to trace.

### Caveats 😖

This script does not do any shuffling / randomisation of the nameservers it pick (and neither the extension does it).
What this means is the script will _always_ (well, most of the time) pick the same set of servers. One case where this
isn't true is when the nameserver does [load balancing](https://www.cloudflare.com/en-in/learning/performance/what-is-dns-load-balancing/).
Most root servers and servers of TLDs would do this.
