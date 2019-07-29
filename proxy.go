package goftp

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

type Direct struct{}

func (Direct) Dial(network, addr string) (net.Conn, error) {
	return net.Dial(network, addr)
}

type HttpProxy struct {
	host     string
	haveAuth bool
	username string
	password string
	forward  proxy.Dialer
}

func NewHTTPProxy(uri *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	s := new(HttpProxy)
	s.host = uri.Host
	s.forward = forward

	return s, nil
}

func (s *HttpProxy) Dial(network, addr string) (net.Conn, error) {
	c, err := s.forward.Dial("tcp", s.host)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse("http://" + addr)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	reqURL.Scheme = ""

	req, err := http.NewRequest("CONNECT", reqURL.String(), nil)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	req.Close = false
	if s.haveAuth {
		req.SetBasicAuth(s.username, s.password)
	}

	err = req.Write(c)
	if err != nil {
		_ = c.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		_ = resp.Body.Close()
		_ = c.Close()
		return nil, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		_ = c.Close()
		err = fmt.Errorf("connect server using proxy error, StatusCode [%d]", resp.StatusCode)
		return nil, err
	}

	return c, nil
}

func GetProxyConn(proxyUrl, destUrl string) (net.Conn, error) {
	uri, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}

	proxy.RegisterDialerType("http", NewHTTPProxy)
	proxy.RegisterDialerType("https", NewHTTPProxy)

	direct := Direct{}
	dialer, err := proxy.FromURL(uri, direct)
	if err != nil {
		return nil, err
	}

	return dialer.Dial("tcp", destUrl)
}
