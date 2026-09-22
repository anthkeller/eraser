package history

import (
	"database/sql"
	"fmt"
	"time"
)

type ExposureStatus string

const (
	ExposureFound      ExposureStatus = "found"
	ExposurePossible   ExposureStatus = "possible"
	ExposureNotFound   ExposureStatus = "not_found"
	ExposureUnknown    ExposureStatus = "unknown"
	ExposureError      ExposureStatus = "error"
	ExposureRemoved    ExposureStatus = "removed"
	ExposureReappeared ExposureStatus = "reappeared"
)

// TransitionExposureStatus turns a raw check into a meaningful monitoring
// event when a record disappears or returns.
func TransitionExposureStatus(previous, current ExposureStatus) ExposureStatus {
	wasPresent := previous == ExposureFound || previous == ExposurePossible || previous == ExposureReappeared
	isPresent := current == ExposureFound || current == ExposurePossible
	if wasPresent && current == ExposureNotFound {
		return ExposureRemoved
	}
	wasAbsent := previous == ExposureNotFound || previous == ExposureRemoved
	if wasAbsent && isPresent {
		return ExposureReappeared
	}
	return current
}

type ExposureCheck struct {
	ID          int64
	BrokerID    string
	BrokerName  string
	Status      ExposureStatus
	RecordURL   string
	Evidence    string
	Confidence  float64
	Error       string
	CheckedAt   time.Time
	NextCheckAt time.Time
}

func (s *Store) AddExposureCheck(check *ExposureCheck) error {
	if check.CheckedAt.IsZero() {
		check.CheckedAt = time.Now().UTC()
	}
	var next any
	if !check.NextCheckAt.IsZero() {
		next = check.NextCheckAt
	}
	result, err := s.db.Exec(`INSERT INTO exposure_checks
		(broker_id, broker_name, status, record_url, evidence, confidence, error, checked_at, next_check_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, check.BrokerID, check.BrokerName, check.Status,
		check.RecordURL, check.Evidence, check.Confidence, check.Error, check.CheckedAt, next)
	if err != nil {
		return fmt.Errorf("failed to insert exposure check: %w", err)
	}
	check.ID, err = result.LastInsertId()
	return err
}

func scanExposure(scanner interface{ Scan(...any) error }) (*ExposureCheck, error) {
	var c ExposureCheck
	var recordURL, evidence, errText sql.NullString
	var next sql.NullTime
	err := scanner.Scan(&c.ID, &c.BrokerID, &c.BrokerName, &c.Status, &recordURL,
		&evidence, &c.Confidence, &errText, &c.CheckedAt, &next)
	if err != nil {
		return nil, err
	}
	c.RecordURL, c.Evidence, c.Error = recordURL.String, evidence.String, errText.String
	if next.Valid {
		c.NextCheckAt = next.Time
	}
	return &c, nil
}

func (s *Store) GetLatestExposureCheck(brokerID string) (*ExposureCheck, error) {
	c, err := scanExposure(s.db.QueryRow(`SELECT id, broker_id, broker_name, status, record_url,
		evidence, confidence, error, checked_at, next_check_at FROM exposure_checks
		WHERE broker_id = ? ORDER BY checked_at DESC, id DESC LIMIT 1`, brokerID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query latest exposure check: %w", err)
	}
	return c, nil
}

func (s *Store) GetLatestExposureChecks(limit int) ([]ExposureCheck, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id, broker_id, broker_name, status, record_url,
		evidence, confidence, error, checked_at, next_check_at FROM exposure_checks
		WHERE id IN (SELECT MAX(id) FROM exposure_checks GROUP BY broker_id)
		ORDER BY checked_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query exposure checks: %w", err)
	}
	defer rows.Close()
	checks := make([]ExposureCheck, 0)
	for rows.Next() {
		check, scanErr := scanExposure(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan exposure check: %w", scanErr)
		}
		checks = append(checks, *check)
	}
	return checks, rows.Err()
}

// GetDueBrokerIDs returns brokers never checked or whose latest check is due.
func (s *Store) GetDueBrokerIDs(brokerIDs []string, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = len(brokerIDs)
	}
	due := make([]string, 0, limit)
	for _, id := range brokerIDs {
		latest, err := s.GetLatestExposureCheck(id)
		if err != nil {
			return nil, err
		}
		if latest == nil || latest.NextCheckAt.IsZero() || !latest.NextCheckAt.After(now) {
			due = append(due, id)
			if len(due) == limit {
				break
			}
		}
	}
	return due, nil
}
