package accounts

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type Account struct {
	Service  string
	LoginURL string
	Username string
	Source   string
}

// ImportCSV reads account identity metadata but deliberately has no password
// field. Password, TOTP, notes, and custom-field columns are never returned.
func ImportCSV(reader io.Reader, source string) ([]Account, error) {
	csvReader := csv.NewReader(io.LimitReader(reader, 50<<20))
	csvReader.FieldsPerRecord = -1
	rows, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read account export: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	headers := map[string]int{}
	for i, header := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(header))] = i
	}
	if source == "" || source == "auto" {
		source = detectSource(headers)
	}
	nameColumns, urlColumns, userColumns := columnsFor(source)
	if firstIndex(headers, urlColumns) < 0 {
		return nil, fmt.Errorf("unsupported account export format")
	}
	result := make([]Account, 0, len(rows)-1)
	seen := map[string]bool{}
	for _, row := range rows[1:] {
		loginURL := cell(row, firstIndex(headers, urlColumns))
		parsed, parseErr := url.Parse(loginURL)
		if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			continue
		}
		username := cell(row, firstIndex(headers, userColumns))
		service := cell(row, firstIndex(headers, nameColumns))
		if service == "" {
			service = parsed.Hostname()
		}
		key := strings.ToLower(parsed.Hostname() + "|" + username)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, Account{Service: service, LoginURL: parsed.String(), Username: username, Source: source})
	}
	return result, nil
}

func detectSource(headers map[string]int) string {
	if _, ok := headers["login_uri"]; ok {
		return "bitwarden"
	}
	if _, ok := headers["login_password"]; ok {
		return "bitwarden"
	}
	if _, ok := headers["title"]; ok {
		return "1password"
	}
	return "browser"
}

func columnsFor(source string) (names, urls, users []string) {
	switch strings.ToLower(source) {
	case "bitwarden":
		return []string{"name"}, []string{"login_uri"}, []string{"login_username"}
	case "1password", "onepassword":
		return []string{"title", "name"}, []string{"url", "website"}, []string{"username", "email"}
	default:
		return []string{"name", "title"}, []string{"url", "origin_url", "website"}, []string{"username", "email"}
	}
}

func firstIndex(headers map[string]int, names []string) int {
	for _, name := range names {
		if index, ok := headers[name]; ok {
			return index
		}
	}
	return -1
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}
