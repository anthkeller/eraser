package discovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eraser-privacy/eraser/internal/broker"
	"github.com/eraser-privacy/eraser/internal/config"
	"github.com/eraser-privacy/eraser/internal/history"
)

const maxResponseBytes = 2 << 20

type Checker interface {
	Name() string
	Supports(broker.Broker) bool
	Check(context.Context, broker.Broker, config.Profile) history.ExposureCheck
}

type Registry struct {
	checkers []Checker
}

func DefaultRegistry(client *http.Client) *Registry {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Registry{checkers: []Checker{&WebChecker{Client: client}}}
}

func (r *Registry) Resolve(b broker.Broker) (Checker, error) {
	wanted := strings.ToLower(b.Workflow.Discovery.Adapter)
	for _, checker := range r.checkers {
		if wanted != "" && wanted != checker.Name() {
			continue
		}
		if checker.Supports(b) {
			return checker, nil
		}
	}
	return nil, fmt.Errorf("broker %q has no supported discovery adapter", b.ID)
}

func (r *Registry) Check(ctx context.Context, b broker.Broker, profile config.Profile) history.ExposureCheck {
	checker, err := r.Resolve(b)
	if err != nil {
		return history.ExposureCheck{BrokerID: b.ID, BrokerName: b.Name, Status: history.ExposureUnknown, Error: err.Error()}
	}
	return checker.Check(ctx, b, profile)
}

type WebChecker struct {
	Client *http.Client
}

func (w *WebChecker) Name() string { return "web" }
func (w *WebChecker) Supports(b broker.Broker) bool {
	t := strings.ToLower(b.Workflow.Discovery.Type)
	a := strings.ToLower(b.Workflow.Discovery.Adapter)
	return b.Workflow.Discovery.URL != "" && (t == "web" || t == "url") && (a == "" || a == w.Name())
}

func (w *WebChecker) Check(ctx context.Context, b broker.Broker, profile config.Profile) history.ExposureCheck {
	check := history.ExposureCheck{BrokerID: b.ID, BrokerName: b.Name, Status: history.ExposureUnknown, CheckedAt: time.Now().UTC()}
	target, err := expandURL(b.Workflow.Discovery.URL, profile)
	if err != nil {
		check.Status, check.Error = history.ExposureError, err.Error()
		return check
	}
	check.RecordURL = target
	method := strings.ToUpper(b.Workflow.Discovery.Method)
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet {
		check.Status, check.Error = history.ExposureError, "discovery currently supports GET requests only"
		return check
	}
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		check.Status, check.Error = history.ExposureError, err.Error()
		return check
	}
	req.Header.Set("User-Agent", "Eraser Privacy Monitor/1.0")
	resp, err := w.Client.Do(req)
	if err != nil {
		check.Status, check.Error = history.ExposureError, err.Error()
		return check
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		check.Status, check.Error = history.ExposureError, fmt.Sprintf("HTTP %d", resp.StatusCode)
		return check
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		check.Status, check.Error = history.ExposureError, err.Error()
		return check
	}
	check.Status, check.Confidence, check.Evidence = classify(string(body), profile)
	return check
}

func expandURL(raw string, p config.Profile) (string, error) {
	replacements := map[string]string{
		"{first_name}": p.FirstName, "{last_name}": p.LastName, "{full_name}": p.FullName(),
		"{city}": p.City, "{state}": p.State, "{zip_code}": p.ZipCode, "{email}": p.Email,
	}
	for token, value := range replacements {
		raw = strings.ReplaceAll(raw, token, url.QueryEscape(value))
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid discovery URL")
	}
	return u.String(), nil
}

func classify(body string, p config.Profile) (history.ExposureStatus, float64, string) {
	text := strings.ToLower(body)
	fullName := strings.ToLower(strings.TrimSpace(p.FullName()))
	if fullName == "" || !strings.Contains(text, fullName) {
		return history.ExposureNotFound, 0.8, "profile name not present in response"
	}
	signals := []string{p.City, p.State, p.ZipCode, p.Email, p.Phone}
	for _, signal := range signals {
		signal = strings.ToLower(strings.TrimSpace(signal))
		if signal != "" && strings.Contains(text, signal) {
			return history.ExposureFound, 0.95, "profile name and an additional identity signal matched"
		}
	}
	return history.ExposurePossible, 0.6, "profile name matched without a second identity signal"
}
