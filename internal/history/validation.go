package history

import (
	"database/sql"
	"fmt"
	"time"
)

type BrokerValidation struct {
	ID                int64
	BrokerID          string
	WebsiteValid      bool
	WebsiteStatusCode int
	OptOutValid       bool
	OptOutStatusCode  int
	EmailDomainValid  bool
	Error             string
	CheckedAt         time.Time
}

func (s *Store) AddBrokerValidation(v *BrokerValidation) error {
	if v.CheckedAt.IsZero() {
		v.CheckedAt = time.Now().UTC()
	}
	result, err := s.db.Exec(`INSERT INTO broker_validations
		(broker_id, website_valid, website_status_code, optout_valid, optout_status_code, email_domain_valid, error, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, v.BrokerID, v.WebsiteValid, v.WebsiteStatusCode,
		v.OptOutValid, v.OptOutStatusCode, v.EmailDomainValid, v.Error, v.CheckedAt)
	if err != nil {
		return fmt.Errorf("failed to insert broker validation: %w", err)
	}
	v.ID, err = result.LastInsertId()
	return err
}

func scanValidation(scanner interface{ Scan(...any) error }) (*BrokerValidation, error) {
	var v BrokerValidation
	var errText sql.NullString
	err := scanner.Scan(&v.ID, &v.BrokerID, &v.WebsiteValid, &v.WebsiteStatusCode,
		&v.OptOutValid, &v.OptOutStatusCode, &v.EmailDomainValid, &errText, &v.CheckedAt)
	v.Error = errText.String
	return &v, err
}

func (s *Store) GetLatestBrokerValidations() (map[string]BrokerValidation, error) {
	rows, err := s.db.Query(`SELECT id, broker_id, website_valid, website_status_code,
		optout_valid, optout_status_code, email_domain_valid, error, checked_at
		FROM broker_validations WHERE id IN (SELECT MAX(id) FROM broker_validations GROUP BY broker_id)`)
	if err != nil {
		return nil, fmt.Errorf("failed to query broker validations: %w", err)
	}
	defer rows.Close()
	result := map[string]BrokerValidation{}
	for rows.Next() {
		v, scanErr := scanValidation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result[v.BrokerID] = *v
	}
	return result, rows.Err()
}
