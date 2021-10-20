package server

import (
	"crypto/tls"
	"fmt"
	"github.com/0xERR0R/dns-proxy/cert"
	"github.com/0xERR0R/dns-proxy/config"
	"github.com/0xERR0R/dns-proxy/doh"
	"github.com/avast/retry-go"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

type Server struct {
	dnsServers     []*dns.Server
	certHolder     cert.Holder
	serverHost     string
	client         *doh.Client
	retryAttempts  uint
	fallbackClient *doh.Client
}

func NewServer(certHolder cert.Holder, cfg config.ProxyConfig) (server *Server, err error) {
	dnsServers := []*dns.Server{
		createTLSServer(certHolder, cfg),
		createUDPServer(),
		createTCPServer(),
	}

	client, err := doh.NewDohClient(cfg.UpstreamTimeout, cfg.UpstreamDOH...)
	if err != nil {
		return nil, fmt.Errorf("can't create DoH client: %w", err)
	}

	fallbackClient, err := doh.NewDohClient(cfg.UpstreamTimeout, cfg.FallbackDOH)
	if err != nil {
		return nil, fmt.Errorf("can't create DoH client: %w", err)
	}

	s := &Server{
		dnsServers:     dnsServers,
		certHolder:     certHolder,
		serverHost:     cfg.TLSDomain,
		client:         client,
		retryAttempts:  cfg.UpstreamRetryAttempts,
		fallbackClient: fallbackClient,
	}
	for _, server := range s.dnsServers {
		handler := server.Handler.(*dns.ServeMux)
		handler.HandleFunc(".", s.OnRequest)
	}

	return s, nil
}

func createTLSServer(h cert.Holder, cfg config.ProxyConfig) *dns.Server {

	return &dns.Server{
		Addr: ":853",
		Net:  "tcp-tls",
		TLSConfig: &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return h.GetCertificate()
			},
			VerifyConnection: func(state tls.ConnectionState) error {
				if strings.HasSuffix(strings.ToLower(state.ServerName), strings.ToLower(cfg.TLSDomain)) {
					return nil
				}
				log.Errorf("name missmatch, unknown request server name '%s'", state.ServerName)
				return fmt.Errorf("name missmatch, unknown request server name '%s'", state.ServerName)
			},
			MinVersion: tls.VersionTLS12,
		},
		Handler: dns.NewServeMux(),
		NotifyStartedFunc: func() {
			log.Infof("TLS server is up and running")
		},
	}
}

func createUDPServer() *dns.Server {
	return &dns.Server{
		Addr:    ":53",
		Net:     "udp",
		Handler: dns.NewServeMux(),
		NotifyStartedFunc: func() {
			log.Infof("UDP server is up and running")
		},
		UDPSize: 65535,
	}
}

func createTCPServer() *dns.Server {
	return &dns.Server{
		Addr:    ":53",
		Net:     "tcp",
		Handler: dns.NewServeMux(),
		NotifyStartedFunc: func() {
			log.Infof("UDP server is up and running")
		},
	}
}

// OnRequest will be executed if a new DNS request is received
func (s *Server) OnRequest(rw dns.ResponseWriter, request *dns.Msg) {
	log.Debugf("new request")

	var hostName string

	var remoteAddr net.Addr

	if rw != nil {
		remoteAddr = rw.RemoteAddr()
	}

	clientIP := resolveClientIP(remoteAddr)
	con, ok := rw.(dns.ConnectionStater)

	if ok && con.ConnectionState() != nil {
		hostName = con.ConnectionState().ServerName
	}

	clientId := strings.ReplaceAll(strings.TrimSuffix(hostName, s.serverHost), ".", "")

	var response *dns.Msg
	err := retry.Do(
		func() error {
			var err error
			response, err = s.client.DoProxyRequest(request, clientIP, clientId)
			if err != nil {
				log.Trace("error occurred during DOH proxy request: ", err)
				return err
			}
			return nil
		},
		retry.Attempts(s.retryAttempts),
	)

	if err != nil {
		log.Error("can't process request, error occurred after all retry attempts, will use fallback now: ", err)
		response, err = s.fallbackClient.DoProxyRequest(request, clientIP, clientId)
	}

	if err != nil {
		log.Error("error occurred during DOH proxy request after fallback, give up...: ", err)
		m := new(dns.Msg)
		m.SetRcode(request, dns.RcodeServerFailure)
		if err := rw.WriteMsg(m); err != nil {
			log.Error("error occurred on sending the server failure response: ", err)
		}
		return

	}

	response.MsgHdr.RecursionAvailable = request.MsgHdr.RecursionDesired

	// truncate if necessary
	response.Truncate(getMaxResponseSize(rw.LocalAddr().Network(), request))

	// enable compression
	response.Compress = true

	err = rw.WriteMsg(response)

	if err != nil {
		log.Error("can't write response: ", err)
	}

}

func resolveClientIP(addr net.Addr) net.IP {
	if t, ok := addr.(*net.UDPAddr); ok {
		return t.IP
	} else if t, ok := addr.(*net.TCPAddr); ok {
		return t.IP
	}

	return nil
}

// returns EDNS upd size or if not present, 512 for UDP and 64K for TCP
func getMaxResponseSize(network string, request *dns.Msg) int {
	edns := request.IsEdns0()
	if edns != nil && edns.UDPSize() > 0 {
		return int(edns.UDPSize())
	}

	if network == "tcp" {
		return dns.MaxMsgSize
	}

	return dns.MinMsgSize
}

// Start starts the server
func (s *Server) Start() {
	log.Info("Starting server")

	for _, srv := range s.dnsServers {
		srv := srv

		go func() {
			if err := srv.ListenAndServe(); err != nil {
				log.Fatalf("start %s listener failed: %v", srv.Net, err)
			}
		}()
	}
}

// Stop stops the server
func (s *Server) Stop() {
	log.Info("Stopping server")

	for _, server := range s.dnsServers {
		if err := server.Shutdown(); err != nil {
			log.Fatalf("stop %s listener failed: %v", server.Net, err)
		}
	}
}
