package config

import (
	"fmt"
	"time"
)

type ProxyConfig struct {
	TLSDomain             string        `env:"TLS_DOMAIN" usage:"Domain name without wildcards. Used to create wildcard certificate and to check incoming connections"`
	PreferredChain        string        `env:"PREFERRED_CHAIN" default:"ISRG Root X1" usage:"preferred certificate chain to use"`
	Email                 string        `env:"EMAIL" usage:"registration email address"`
	DNSProvider           string        `env:"DNS_PROVIDER" usage:"One of supported provider from https://go-acme.github.io/lego/dns/ "`
	CertDir               string        `env:"CERT_DIR" default:"./certs" usage:"Directory for generated certificates"`
	LogLevel              string        `env:"LOG_LEVEL" default:"info" usage:"Level to log. One of 'trace, debug, info, warn, error, fatal'"`
	UpstreamDOH           []string      `env:"UPSTREAM_DOH" default:"https://cloudflare-dns.com/dns-query" usage:"Comma separated list of upstream DoH DNS resolvers"`
	UpstreamRetryAttempts uint          `env:"UPSTREAM_RETRY_CNT" default:"2" usage:"Number of retry attempts before fallback resolver will be invoked"`
	FallbackDOH           string        `env:"FALLBACK_DOH" default:"https://cloudflare-dns.com/dns-query" usage:"Fallback upstream DoH server, used if upstream DoH requests fail"`
	UpstreamTimeout       time.Duration `env:"UPSTREAM_TIMEOUT" default:"1s" usage:"timeout for the upstream DoH request"`
	RenewThresholdDays    uint          `env:"RENEW_THRESHOLD_DAYS" default:"7" usage:"Renew certificate if it expires in X or less days"`
}

func (p ProxyConfig) String() string {
	return fmt.Sprintf(`  TLSDomain: %s
  PreferredChain: %s
  Email: %s
  DNSProvider: %s
  CertDir: %s
  LogLevel: %s
  UpstreamDOH: %s
  UpstreamRetryAttempts: %d
  RenewThresholdDays: %d
  FallbackDOH: %s
  UpstreamTimeout: %s`,
		p.TLSDomain, p.PreferredChain,
		p.Email, p.DNSProvider,
		p.CertDir, p.LogLevel,
		p.UpstreamDOH, p.UpstreamRetryAttempts,
		p.RenewThresholdDays, p.FallbackDOH,
		p.UpstreamTimeout)
}
