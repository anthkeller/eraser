package accounts

import (
	"reflect"
	"strings"
	"testing"
)

func TestImportBrowserCSVExcludesSecrets(t *testing.T) {
	csv := "name,url,username,password,note\nExample,https://example.com/login,jane@example.com,super-secret,private note\n"
	accounts, err := ImportCSV(strings.NewReader(csv), "auto")
	if err != nil {
		t.Fatal(err)
	}
	expected := []Account{{Service: "Example", LoginURL: "https://example.com/login", Username: "jane@example.com", Source: "browser"}}
	if !reflect.DeepEqual(accounts, expected) {
		t.Fatalf("unexpected accounts: %#v", accounts)
	}
	if strings.Contains(strings.ToLower(strings.Join([]string{accounts[0].Service, accounts[0].LoginURL, accounts[0].Username}, "|")), "super-secret") {
		t.Fatal("password leaked")
	}
}

func TestImportBitwardenAndDeduplicate(t *testing.T) {
	csv := "name,login_uri,login_username,login_password\nExample,https://example.com,jane,pw\nDuplicate,https://example.com/path,jane,pw2\n"
	accounts, err := ImportCSV(strings.NewReader(csv), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].Source != "bitwarden" {
		t.Fatalf("unexpected accounts: %#v", accounts)
	}
}
