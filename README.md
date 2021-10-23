# dns-proxy

## Configuration

| Variable        | Example             | Description                                              |
|-----------------|---------------------|----------------------------------------------------------|
| TLS_DOMAIN      | my.duckdns.org      | Domain name without wildcards. Used to create wildcard certificate and to check incoming connections |
| DNS_PROVIDER    | duckdns             | One of supported provider from https://go-acme.github.io/lego/dns/ |
| DUCKDNS_TOKEN   | 1fff-111-...        | Provider specific token, see https://go-acme.github.io/lego/dns/ for further information |
| CERT_DIR        | /opt/certs          | Directory for generated certificates. Default: ./certs             |
| EMAIL           | me@me.com           | Registration email address for Let's Encrypt|
| LOG_LEVEL       | debug               | Level to log. One of 'trace, debug, info, warn, error, fatal'. Default: info|
| PREFERRED_CHAIN | ISRG Root X1        | preferred certificate chain to use. default: "ISRG Root X1" |
| UPSTREAM_DOH    | http://192.168.178.3:4000/dns-query,https://cloudflare-dns.com/dns-query | Comma separated list of upstream DoH DNS resolvers. Placeholder `_CLIENTID_` will be automatically replaced with client id (only TLS from subdomain) |
| UPSTREAM_RETRY_CNT    | 2 | Number of retry attempts before fallback resolver will be invoked. Default: 2 |
| UPSTREAM_TIMEOUT    | 1s | timeout for the upstream DoH request. Default: 1s |
| FALLBACK_DOH    | https://cloudflare-dns.com/dns-query | Fallback upstream DoH server, used if upstream DoH requests fail. Default: https://cloudflare-dns.com/dns-query  |

## Example with docker-compose and blocky

dns-proxy as DoT with Let's encrypt certificate and duckdns domain "test.duckdns.org". Uses 2 blocky instances as DoH
resolver (192.168.178.3 and 192.168.178.5). Clients can use either "test.duckdns.org" for anonymous or "
XXX.test.duckdns.org" for named access (in this case XXX will be passed to blocky for logging and filtering purposes).

```yaml
version: "2.1"
services:
  dns-proxy:
    image: ghcr.io/0xerr0r/dns-proxy
    container_name: dns-proxy
    restart: always
    ports:
      - "853:853"
      - "53:53"
    environment:
      - TZ=Europe/Berlin
      - TLS_DOMAIN=test.duckdns.org
      - DNS_PROVIDER=duckdns
      - DUCKDNS_TOKEN=1df927c4-YOUR_TOKEN_HERE-XXX
      - EMAIL=your@email.here
      - LOG_LEVEL=info
      - UPSTREAM_DOH=http://192.168.178.3:4000/dns-query/_CLIENTID_,http://192.168.178.5:4000/dns-query/_CLIENTID_
    volumes:
      - certs:/app/certs
volumes:
  certs:
```
