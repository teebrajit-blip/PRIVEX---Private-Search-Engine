package urlnorm

import (
	"net"
	"net/url"
	"strings"
)

var trackingParams = map[string]struct{}{
	"fbclid": {}, "gclid": {}, "dclid": {}, "mc_cid": {}, "mc_eid": {},
}

// Normalize conservatively lowercases scheme and host, removes fragments and
// default ports, sorts query parameters, and removes common tracking keys.
func Normalize(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", &url.Error{Op: "normalize", URL: raw, Err: net.InvalidAddrError("unsupported URL scheme")}
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	host := u.Hostname()
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		u.Host = host
	}
	u.User = nil

	query := u.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if lower == "utm_source" || lower == "utm_medium" || lower == "utm_campaign" ||
			lower == "utm_term" || lower == "utm_content" || lower == "utm_id" {
			query.Del(key)
			continue
		}
		if _, ok := trackingParams[lower]; ok {
			query.Del(key)
		}
	}
	u.RawQuery = query.Encode()
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}
