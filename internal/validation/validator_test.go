package validation

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eraser-privacy/eraser/internal/broker"
)

type fakeResolver struct {
	private bool
	mx      bool
}

func (r fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	ip := net.ParseIP("203.0.113.10")
	if r.private {
		ip = net.ParseIP("127.0.0.1")
	}
	return []net.IPAddr{{IP: ip}}, nil
}
func (r fakeResolver) LookupMX(context.Context, string) ([]*net.MX, error) {
	if !r.mx {
		return nil, nil
	}
	return []*net.MX{{Host: "mail.example.com", Pref: 10}}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestValidatePublicEndpointsAndMX(t *testing.T) {
	client := &http.Client{Timeout: time.Second, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}, Request: r}, nil
	})}
	v := &Validator{Client: client, Resolver: fakeResolver{mx: true}}
	result := v.Validate(context.Background(), broker.Broker{ID: "example", Email: "privacy@example.com", Website: "https://example.com", OptOutURL: "https://example.com/optout"})
	if !result.WebsiteValid || !result.OptOutValid || !result.EmailDomainValid {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRejectsPrivateDestinations(t *testing.T) {
	v := &Validator{Client: &http.Client{Timeout: time.Second}, Resolver: fakeResolver{private: true}}
	result := v.Validate(context.Background(), broker.Broker{ID: "example", Website: "http://internal.example"})
	if result.WebsiteValid || !strings.Contains(result.Error, "non-public") {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestReachableBotProtectionStatuses(t *testing.T) {
	for _, status := range []int{200, 301, 401, 403, 429} {
		if !reachable(status) {
			t.Fatalf("status %d should be reachable", status)
		}
	}
	for _, status := range []int{404, 410, 500} {
		if reachable(status) {
			t.Fatalf("status %d should not be reachable", status)
		}
	}
}
