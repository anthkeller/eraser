package intelligence

import (
	"sort"
	"strings"

	"github.com/eraser-privacy/eraser/internal/broker"
)

type ValidationSignals struct {
	WebsiteValid     bool
	OptOutValid      bool
	EmailDomainValid bool
	Checked          bool
}

type Score struct {
	Quality  int
	Risk     int
	Priority int
	Reasons  []string
}

var riskWeights = map[string]int{
	"account-credentials": 25,
	"government-id":       25,
	"reproductive-health": 20,
	"biometric":           20,
	"precise-geolocation": 15,
	"minors":              15,
	"date-of-birth":       10,
}

func Calculate(b broker.Broker, validation ValidationSignals) Score {
	quality := 0
	reasons := make([]string, 0)
	if b.Email != "" {
		quality += 20
	}
	if b.Website != "" {
		quality += 10
	}
	if b.OptOutURL != "" {
		quality += 20
	}
	if len(b.Sources) > 0 {
		quality += 15
	}
	if len(b.Sources) > 1 {
		quality += 15
		reasons = append(reasons, "confirmed by multiple registries")
	}
	if len(b.RegistryIDs) > 0 {
		quality += 10
	}
	if validation.Checked {
		if validation.WebsiteValid {
			quality += 5
		} else if b.Website != "" {
			quality -= 10
			reasons = append(reasons, "website validation failed")
		}
		if validation.OptOutValid {
			quality += 10
		} else if b.OptOutURL != "" {
			quality -= 15
			reasons = append(reasons, "opt-out validation failed")
		}
		if validation.EmailDomainValid {
			quality += 10
		} else if b.Email != "" {
			quality -= 20
			reasons = append(reasons, "email domain validation failed")
		}
	}
	quality = clamp(quality)

	risk := 0
	flags := append([]string(nil), b.RiskFlags...)
	sort.Strings(flags)
	for _, flag := range flags {
		risk += riskWeights[strings.ToLower(flag)]
	}
	risk = clamp(risk)
	if risk > 0 {
		reasons = append(reasons, "sensitive data: "+strings.Join(flags, ", "))
	}
	priority := clamp((risk*65 + quality*35) / 100)
	return Score{Quality: quality, Risk: risk, Priority: priority, Reasons: reasons}
}

func clamp(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
