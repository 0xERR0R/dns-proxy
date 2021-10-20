package cert

import (
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"time"
)

type Holder interface {
	GetCertificate() (*tls.Certificate, error)
}

type RefreshingCertHolder struct {
	manager Manager
	current *tls.Certificate
}

func (r *RefreshingCertHolder) GetCertificate() (*tls.Certificate, error) {
	return r.current, nil
}

func NewRefreshingCertHolder(manager Manager) Holder {
	h := &RefreshingCertHolder{manager: manager}
	h.update()
	go periodicUpdate(h)
	return h

}

func periodicUpdate(h *RefreshingCertHolder) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		<-ticker.C
		h.update()
	}
}

func (r *RefreshingCertHolder) update() {
	log.Tracef("updating cert")
	// TODO retry here
	certificate, err := r.manager.RetrieveCertificate()
	if err != nil {
		log.Error("can't update certificate: ", err)
	} else {
		r.current = certificate
	}
}
