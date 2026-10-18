package egress

import (
	"net"
	"net/url"
)

// splitHostPort is a thin wrapper around net.SplitHostPort that
// returns the input untouched on error. We use this in normaliseHost
// because we want a no-op for inputs that are not in the
// "host:port" or "[ipv6]:port" form.
func splitHostPort(h string) (host, port string, err error) {
	return net.SplitHostPort(h)
}

// extractHost pulls the host (and port) out of a URL string. An
// unparsable URL returns the empty string. The returned host is the
// raw value the URL parser produced; normaliseHost lower-cases it
// for comparison.
func extractHost(rawURL string) (host string, err error) {
	if rawURL == "" {
		return "", &url.Error{Op: "extractHost", URL: rawURL, Err: errEmptyURL}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

// errEmptyURL is the sentinel returned when the URL is empty. It is
// a tiny value type so we don't pull in errors.New at call sites.
type emptyURLError struct{}

func (emptyURLError) Error() string { return "empty url" }

var errEmptyURL = emptyURLError{}
