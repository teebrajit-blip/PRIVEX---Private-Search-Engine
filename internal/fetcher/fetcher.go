package fetcher

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrUnsafeURL = errors.New("URL resolves to a blocked address")

type Client struct {
	http     *http.Client
	maxBytes int64
}

func New(maxBytes int64) *Client {
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return validateURL(req.URL)
			},
		},
		maxBytes: maxBytes,
	}
}

func (c *Client) Fetch(ctx context.Context, rawURL string) (*http.Response, []byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, nil, ErrUnsafeURL
	}
	if err := validateURL(u); err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "PRIVEXBot/1.0 (+privacy-first focused crawler)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") {
		return resp, nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(body)) > c.maxBytes {
		return nil, nil, errors.New("response exceeds size limit")
	}
	return resp, body, nil
}

func validateURL(u *url.URL) error {
	host := u.Hostname()
	if host == "" {
		return ErrUnsafeURL
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return ErrUnsafeURL
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return ErrUnsafeURL
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return ErrUnsafeURL
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 169 && ip4[1] == 254
	}
	return false
}
