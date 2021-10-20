package doh

import (
	"bytes"
	"fmt"
	roundrobin "github.com/hlts2/round-robin"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	dnsContentType     = "application/dns-message"
	clientIdPattern    = "_CLIENTID_"
	forwardedForHeader = "X-FORWARDED-FOR"
	contentTypeHeader  = "content-type"
)

type Client struct {
	httpClient *http.Client
	rr         roundrobin.RoundRobin
}

func NewDohClient(timeout time.Duration, urls ...string) (*Client, error) {
	httpClient := &http.Client{
		Timeout: timeout,
	}

	rr, err := createRoundRobin(urls)
	if err != nil {
		return nil, fmt.Errorf("can't create round robin: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		rr:         rr,
	}, nil
}

func createRoundRobin(u []string) (roundrobin.RoundRobin, error) {
	var urls []*url.URL

	for _, doh := range u {
		doh = strings.TrimSpace(doh)
		u, err := url.Parse(doh)
		if err == nil && u.Scheme != "" && u.Host != "" {
			urls = append(urls, u)
		} else {
			return nil, fmt.Errorf("wrong DoH url '%s': %w", doh, err)
		}
	}

	return roundrobin.New(urls...)
}

func (c *Client) DoProxyRequest(request *dns.Msg, ip net.IP, id string) (*dns.Msg, error) {
	log.Debugf("performing DoH request for ip '%s' and clientID '%s'", ip, id)
	rawDNSMessage, err := request.Pack()

	if err != nil {
		return nil, fmt.Errorf("can't pack message: %w", err)
	}

	upstream := strings.ReplaceAll(c.rr.Next().String(), clientIdPattern, id)
	log.Tracef("using '%s' as upstream URL", upstream)

	req, err := http.NewRequest("POST", upstream, bytes.NewReader(rawDNSMessage))
	if err != nil {
		return nil, fmt.Errorf("can't create POST request: %w", err)
	}
	req.Header.Set(forwardedForHeader, ip.String())
	req.Header.Set(contentTypeHeader, dnsContentType)
	httpResponse, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("can't perform https request: %w", err)
	}

	defer func() {
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http return code should be %d, but received %d", http.StatusOK, httpResponse.StatusCode)
	}

	contentType := httpResponse.Header.Get(contentTypeHeader)
	if contentType != dnsContentType {
		return nil, fmt.Errorf("http return content type should be '%s', but was '%s'",
			dnsContentType, contentType)
	}

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	response := new(dns.Msg)
	err = response.Unpack(body)

	if err != nil {
		return nil, fmt.Errorf("can't unpack message: %w", err)
	}

	return response, nil

}
