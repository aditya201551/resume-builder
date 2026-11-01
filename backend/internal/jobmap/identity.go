package jobmap

import (
	"net/url"
	"strings"
)

// ApplicationKey groups original application URLs without changing the link
// used for attribution or navigation. Company and title are not job identities.
func ApplicationKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	query := u.Query()
	for k := range query {
		if strings.HasPrefix(strings.ToLower(k), "utm_") {
			query.Del(k)
		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}
