package GoWinHttp

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WinHttp struct {
	method  string
	url     string
	headers map[string]string
	timeout time.Duration
}

func (w *WinHttp) Open(method, rawURL string) {
	w.method = method
	w.url = rawURL
	w.headers = make(map[string]string)
}

func (w *WinHttp) SetOutTime(connect, send, receive int) {
	total := connect + send + receive
	if total <= 0 {
		total = 30000
	}
	w.timeout = time.Duration(total) * time.Millisecond
}

func (w *WinHttp) SetHeader(key, value string) {
	if w.headers == nil {
		w.headers = make(map[string]string)
	}
	w.headers[key] = value
}

func (w *WinHttp) Send(body string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(w.method, w.url, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{
		Timeout: w.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client.Do(req)
}

type Proxy struct {
	Address  string
	User     string
	Password string
}

type GoWinHttpClient struct {
	useProxy bool
	proxyIP  string
	client   *http.Client
}

func NewGoWinHttp() *GoWinHttpClient {
	return &GoWinHttpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (g *GoWinHttpClient) SetProxyType(useProxy bool) {
	g.useProxy = useProxy
	g.rebuildTransport()
}

func (g *GoWinHttpClient) SetProxyIP(proxyAddr string) {
	g.proxyIP = proxyAddr
	g.rebuildTransport()
}

func (g *GoWinHttpClient) rebuildTransport() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if g.useProxy && g.proxyIP != "" {
		addr := g.proxyIP
		if !strings.Contains(addr, "://") {
			addr = "http://" + addr
		}
		proxyURL, err := url.Parse(addr)
		if err == nil {
			tr.Proxy = http.ProxyURL(proxyURL)
		}
	}
	g.client.Transport = tr
}

func (g *GoWinHttpClient) Do(req *http.Request) (*http.Response, error) {
	return g.client.Do(req)
}

func ConnectS5(conn *net.Conn, proxy *Proxy, host string, port uint16) bool {
	c := *conn
	c.SetDeadline(time.Now().Add(10 * time.Second))
	defer c.SetDeadline(time.Time{})

	authMethod := byte(0x00)
	hasAuth := proxy != nil && proxy.User != "" && proxy.Password != ""
	if hasAuth {
		authMethod = 0x02
	}

	_, err := c.Write([]byte{0x05, 0x01, authMethod})
	if err != nil {
		return false
	}

	buf := make([]byte, 2)
	if _, err = io.ReadFull(c, buf); err != nil {
		return false
	}
	if buf[0] != 0x05 {
		return false
	}

	if buf[1] == 0x02 && hasAuth {
		authReq := []byte{0x01}
		authReq = append(authReq, byte(len(proxy.User)))
		authReq = append(authReq, []byte(proxy.User)...)
		authReq = append(authReq, byte(len(proxy.Password)))
		authReq = append(authReq, []byte(proxy.Password)...)
		if _, err = c.Write(authReq); err != nil {
			return false
		}
		authResp := make([]byte, 2)
		if _, err = io.ReadFull(c, authResp); err != nil {
			return false
		}
		if authResp[1] != 0x00 {
			return false
		}
	}

	req := []byte{0x05, 0x01, 0x00, 0x03}
	req = append(req, byte(len(host)))
	req = append(req, []byte(host)...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))
	req = append(req, portBytes...)

	if _, err = c.Write(req); err != nil {
		return false
	}

	resp := make([]byte, 4)
	if _, err = io.ReadFull(c, resp); err != nil {
		return false
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		return false
	}

	switch resp[3] {
	case 0x01:
		skip := make([]byte, 4+2)
		_, err = io.ReadFull(c, skip)
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err = io.ReadFull(c, lenBuf); err != nil {
			return false
		}
		skip := make([]byte, int(lenBuf[0])+2)
		_, err = io.ReadFull(c, skip)
	case 0x04:
		skip := make([]byte, 16+2)
		_, err = io.ReadFull(c, skip)
	default:
		return false
	}
	if err != nil {
		return false
	}

	_ = fmt.Sprintf("SOCKS5 connected to %s:%d", host, port)
	return true
}
