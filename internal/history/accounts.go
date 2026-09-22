package history

import (
	"fmt"
	"time"

	"github.com/eraser-privacy/eraser/internal/accounts"
)

func (s *Store) UpsertAccount(account accounts.Account) error {
	_, err := s.db.Exec(`INSERT INTO account_inventory (service, login_url, username, source, status, imported_at)
		VALUES (?, ?, ?, ?, 'pending', ?)
		ON CONFLICT(login_url, username) DO UPDATE SET service=excluded.service, source=excluded.source,
		imported_at=excluded.imported_at`, account.Service, account.LoginURL, account.Username, account.Source, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to store account inventory: %w", err)
	}
	return nil
}

func (s *Store) GetAccountInventory(limit int) ([]accounts.Account, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.db.Query(`SELECT service, login_url, username, source FROM account_inventory ORDER BY service LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]accounts.Account, 0)
	for rows.Next() {
		var account accounts.Account
		if err := rows.Scan(&account.Service, &account.LoginURL, &account.Username, &account.Source); err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}
