package validation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/history"
)

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
	LookupMX(context.Context, string) ([]*net.MX, error)
}

type Validator struct {
	Client   *http.Client
	Resolver Resolver
}

func New(timeout time.Duration) *Validator {
	return &Validator{Client: &http.Client{Timeout: timeout}, Resolver: net.DefaultResolver}
}

func (v *Validator) Validate(ctx context.Context, b broker.Broker) history.BrokerValidation {
	result := history.BrokerValidation{BrokerID: b.ID, CheckedAt: time.Now().UTC()}
	errors := make([]string, 0)
	if b.Website != "" {
		result.WebsiteValid, result.WebsiteStatusCode, errors = v.checkURL(ctx, "website", b.Website, errors)
	}
	if b.OptOutURL != "" {
		result.OptOutValid, result.OptOutStatusCode, errors = v.checkURL(ctx, "opt-out", b.OptOutURL, errors)
	}
	if b.Email != "" {
		parts := strings.Split(b.Email, "@")
		if len(parts) == 2 && parts[1] != "" {
			mx, err := v.Resolver.LookupMX(ctx, parts[1])
			result.EmailDomainValid = err == nil && len(mx) > 0
			if err != nil {
				errors = append(errors, "email domain: "+err.Error())
			}
		} else {
			errors = append(errors, "email domain: invalid address")
		}
	}
	result.Error = strings.Join(errors, "; ")
	return result
}

func (v *Validator) checkURL(ctx context.Context, label, raw string, errors []string) (bool, int, []string) {
	if err := v.ensurePublicURL(ctx, raw); err != nil {
		return false, 0, append(errors, label+": "+err.Error())
	}
	client := *v.Client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return v.ensurePublicURL(req.Context(), req.URL.String())
	}
	status, err := requestStatus(ctx, &client, http.MethodHead, raw)
	if err == nil && status == http.StatusMethodNotAllowed {
		status, err = requestStatus(ctx, &client, http.MethodGet, raw)
	}
	if err != nil {
		return false, status, append(errors, label+": "+err.Error())
	}
	return reachable(status), status, errors
}

func requestStatus(ctx context.Context, client *http.Client, method, raw string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Eraser Contact Validator/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func reachable(status int) bool {
	return (status >= 200 && status < 400) || status == 401 || status == 403 || status == 429
}

func (v *Validator) ensurePublicURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("invalid HTTP URL")
	}
	if strings.EqualFold(u.Hostname(), "localhost") {
		return fmt.Errorf("local addresses are not allowed")
	}
	addresses, err := v.Resolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("DNS returned no addresses")
	}
	for _, address := range addresses {
		ip := address.IP
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() {
			return fmt.Errorf("non-public destination is not allowed")
		}
	}
	return nil
}
