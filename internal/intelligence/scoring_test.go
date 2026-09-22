package intelligence

import (
	"testing"

	"github.com/eraser-privacy/eraser/internal/broker"
)

func TestCalculateRewardsProvenanceAndValidation(t *testing.T) {
	b := broker.Broker{Email: "privacy@example.com", Website: "https://example.com", OptOutURL: "https://example.com/optout", Sources: []broker.RegistrySource{{Registry: "ca"}, {Registry: "or"}}, RegistryIDs: map[string]string{"or": "1"}}
	score := Calculate(b, ValidationSignals{Checked: true, WebsiteValid: true, OptOutValid: true, EmailDomainValid: true})
	if score.Quality != 100 {
		t.Fatalf("expected quality 100, got %d", score.Quality)
	}
}

func TestCalculateRiskAndFailedValidation(t *testing.T) {
	b := broker.Broker{Email: "privacy@example.com", Website: "https://example.com", OptOutURL: "https://example.com/optout", RiskFlags: []string{"government-id", "biometric", "precise-geolocation"}}
	score := Calculate(b, ValidationSignals{Checked: true})
	if score.Risk != 60 {
		t.Fatalf("expected risk 60, got %d", score.Risk)
	}
	if score.Quality != 5 {
		t.Fatalf("expected quality 5, got %d", score.Quality)
	}
	if score.Priority != 40 {
		t.Fatalf("expected priority 40, got %d", score.Priority)
	}
}
