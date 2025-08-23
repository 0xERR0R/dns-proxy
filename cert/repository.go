package cert

import (
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

type Repository interface {
	loadCert(domain string) (*tls.Certificate, error)
	storeCert(domain string, certPem []byte, keyPem []byte) error
}

type FileRepository struct {
	dir string
}

func NewFileRepo(dir string) Repository {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			log.Fatalf("can't create certificate directory '%s': %v", dir, err)
		}
	}
	return &FileRepository{dir: dir}
}

func (f *FileRepository) getFileName(domain string) (cert string, key string) {
	return fmt.Sprintf("%s/%s.cert.pem", f.dir, domain), fmt.Sprintf("%s/%s.key.pem", f.dir, domain)
}

func (f *FileRepository) storeCert(domain string, certPem []byte, keyPem []byte) error {
	certFile, keyFile := f.getFileName(domain)
	log.Tracef("storing cert for domain '%s' into '%s'", domain, certFile)

	err := ioutil.WriteFile(certFile, certPem, 0600)
	if err != nil {
		return fmt.Errorf("error on writing cert file: %w", err)
	}
	err = ioutil.WriteFile(keyFile, keyPem, 0600)
	if err != nil {
		return fmt.Errorf("error on writing key file: %w", err)
	}
	log.Tracef("file stored successfully")
	return nil
}

func (f *FileRepository) loadCert(domain string) (*tls.Certificate, error) {
	certFile, keyFile := f.getFileName(domain)
	log.Tracef("loading cert files for domain '%s'", domain)

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error on loading the certificate files: %w", err)
	}

	return &pair, nil
}
