package product

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Edition string
type Deployment string

const (
	EditionCommunity  Edition = "community"
	EditionCloud      Edition = "cloud"
	EditionCommercial Edition = "commercial"

	DeploymentSelfHosted Deployment = "self-hosted"
	DeploymentHosted     Deployment = "hosted"
)

var Version = "dev"

type Capabilities struct {
	SingleUser       bool `json:"single_user"`
	SelfHosting      bool `json:"self_hosting"`
	ManagedService   bool `json:"managed_service"`
	CommercialUse    bool `json:"commercial_use"`
	BrowserExtension bool `json:"browser_extension"`
}

type Settings struct {
	Edition          Edition
	Deployment       Deployment
	BindAddress      string
	PublicURL        string
	OpenBrowser      bool
	OwnerID          string
	AuthIssuer       string
	AuthAudience     string
	AuthClientSecret string
	SessionKey       string
	DatabaseURL      string
}

type PublicInfo struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Edition      Edition      `json:"edition"`
	Deployment   Deployment   `json:"deployment"`
	Capabilities Capabilities `json:"capabilities"`
}

func LoadFromEnv() (Settings, error) {
	settings := Settings{
		Edition:          Edition(valueOrDefault("ERASER_EDITION", string(EditionCommunity))),
		Deployment:       Deployment(valueOrDefault("ERASER_DEPLOYMENT", string(DeploymentSelfHosted))),
		BindAddress:      valueOrDefault("ERASER_BIND", "127.0.0.1"),
		PublicURL:        strings.TrimSpace(os.Getenv("ERASER_PUBLIC_URL")),
		OwnerID:          valueOrDefault("ERASER_OWNER_ID", "local"),
		AuthIssuer:       strings.TrimSpace(os.Getenv("ERASER_AUTH_ISSUER")),
		AuthAudience:     strings.TrimSpace(os.Getenv("ERASER_AUTH_AUDIENCE")),
		AuthClientSecret: strings.TrimSpace(os.Getenv("ERASER_AUTH_CLIENT_SECRET")),
		SessionKey:       strings.TrimSpace(os.Getenv("ERASER_SESSION_KEY")),
		DatabaseURL:      strings.TrimSpace(os.Getenv("ERASER_DATABASE_URL")),
		OpenBrowser:      true,
	}
	if raw, ok := os.LookupEnv("ERASER_OPEN_BROWSER"); ok {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Settings{}, fmt.Errorf("ERASER_OPEN_BROWSER must be true or false")
		}
		settings.OpenBrowser = value
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (s Settings) Validate() error {
	switch s.Edition {
	case EditionCommunity, EditionCloud, EditionCommercial:
	default:
		return fmt.Errorf("unsupported Eraser edition %q", s.Edition)
	}
	switch s.Deployment {
	case DeploymentSelfHosted, DeploymentHosted:
	default:
		return fmt.Errorf("unsupported Eraser deployment %q", s.Deployment)
	}
	if s.Deployment == DeploymentHosted && s.Edition == EditionCommunity {
		return fmt.Errorf("the community edition cannot be configured as a hosted service")
	}
	if strings.TrimSpace(s.BindAddress) == "" {
		return fmt.Errorf("bind address cannot be empty")
	}
	if strings.TrimSpace(s.OwnerID) == "" {
		return fmt.Errorf("owner ID cannot be empty")
	}
	if s.DatabaseURL != "" && s.OwnerID == "local" {
		return fmt.Errorf("PostgreSQL deployments require an explicit ERASER_OWNER_ID")
	}
	if s.Deployment == DeploymentHosted {
		if s.OwnerID == "local" {
			return fmt.Errorf("hosted deployments require an explicit ERASER_OWNER_ID matching the OIDC subject")
		}
		if s.AuthIssuer == "" || s.AuthAudience == "" || s.AuthClientSecret == "" {
			return fmt.Errorf("hosted deployments require ERASER_AUTH_ISSUER, ERASER_AUTH_AUDIENCE, and ERASER_AUTH_CLIENT_SECRET")
		}
		key, err := base64.RawURLEncoding.DecodeString(s.SessionKey)
		if err != nil || len(key) != 64 {
			return fmt.Errorf("hosted deployments require ERASER_SESSION_KEY as 64 random bytes encoded with unpadded base64url")
		}
		if s.DatabaseURL == "" {
			return fmt.Errorf("hosted deployments require ERASER_DATABASE_URL")
		}
		if s.PublicURL == "" {
			return fmt.Errorf("hosted deployments require ERASER_PUBLIC_URL")
		}
	}
	if s.PublicURL != "" {
		parsed, err := url.Parse(s.PublicURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("ERASER_PUBLIC_URL must be an absolute http or https URL")
		}
		if s.Deployment == DeploymentHosted && parsed.Scheme != "https" {
			return fmt.Errorf("hosted deployments require an https ERASER_PUBLIC_URL")
		}
	}
	return nil
}

func (s Settings) Info() PublicInfo {
	return PublicInfo{
		Name:       "Eraser",
		Version:    Version,
		Edition:    s.Edition,
		Deployment: s.Deployment,
		Capabilities: Capabilities{
			SingleUser:       true,
			SelfHosting:      s.Deployment == DeploymentSelfHosted,
			ManagedService:   s.Deployment == DeploymentHosted,
			CommercialUse:    s.Edition == EditionCloud || s.Edition == EditionCommercial,
			BrowserExtension: true,
		},
	}
}

func (s Settings) SecureCookies() bool {
	parsed, err := url.Parse(s.PublicURL)
	return err == nil && parsed.Scheme == "https"
}

func (s Settings) TrustedOrigins(port int) []string {
	origins := []string{"localhost", "127.0.0.1", fmt.Sprintf("localhost:%d", port), fmt.Sprintf("127.0.0.1:%d", port)}
	if parsed, err := url.Parse(s.PublicURL); err == nil && parsed.Host != "" {
		origins = append(origins, parsed.Host)
	}
	return origins
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
