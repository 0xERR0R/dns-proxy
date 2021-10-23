package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/0xERR0R/dns-proxy/config"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Manager interface {
	RetrieveCertificate() (*tls.Certificate, error)
}

type LegoManager struct {
	repo Repository
	cfg  config.ProxyConfig
}

func NewLegoManager(repo Repository, cfg config.ProxyConfig) Manager {
	return &LegoManager{repo, cfg}

}

func (l *LegoManager) RetrieveCertificate() (*tls.Certificate, error) {
	log.Trace("retrieve certificate")
	cert, err := l.repo.loadCert(l.cfg.TLSDomain)
	if err == nil {
		log.Debugf("certificate successful loaded")

		for _, c := range cert.Certificate {
			certificate, _ := x509.ParseCertificate(c)
			if !certificate.IsCA {
				err := certificate.VerifyHostname(l.cfg.TLSDomain)
				if err != nil {
					log.Errorf("certificate is not valid")
				}
				if certificate.NotAfter.Before(time.Now()) {
					log.Errorf("certificate expired at %s", certificate.NotAfter)
				}
				expiresInDays := int(certificate.NotAfter.Sub(time.Now()).Hours() / 24)
				log.Infof("certificate expiration date: %s, expires in %d days", certificate.NotAfter, expiresInDays)
				return cert, nil
			}
		}

	}

	log.Infof("generating new certificates")

	// generate
	domains := []string{l.cfg.TLSDomain, fmt.Sprintf("*.%s", l.cfg.TLSDomain)}
	log.Infof("generating certificates for domains '%s'", strings.Join(domains, ", "))

	client, err := createLegoClient(l.cfg)
	if err != nil {
		return nil, fmt.Errorf("can't create lego client: %w", err)
	}

	request := certificate.ObtainRequest{
		Domains:        domains,
		Bundle:         true,
		PreferredChain: l.cfg.PreferredChain,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("can't obtain certificate: %w", err)
	}

	log.Debugf("retrieved new certificates")

	err = l.repo.storeCert(l.cfg.TLSDomain, certificates.Certificate, certificates.PrivateKey)
	if err != nil {
		log.Error("can't store retrieved certificate: ", err)
	}

	keyPair, err := tls.X509KeyPair(certificates.Certificate, certificates.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("can't create key pair: %w", err)
	}

	return &keyPair, nil
}

func createLegoClient(cfg config.ProxyConfig) (*lego.Client, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("can't generate private key for registration: %w", err)
	}
	myUser := MyUser{
		Email: cfg.Email,
		key:   privateKey,
	}
	config := lego.NewConfig(&myUser)
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("can't create client instance: %w", err)
	}

	provider, err := dns.NewDNSChallengeProviderByName(cfg.DNSProvider)
	if err != nil {
		return nil, fmt.Errorf("can't resolve dns challenge provider: %w", err)
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return nil, fmt.Errorf("can't set dns challenge provider: %w", err)
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("client registration failed: %w", err)
	}
	myUser.Registration = reg
	return client, nil
}

type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}
