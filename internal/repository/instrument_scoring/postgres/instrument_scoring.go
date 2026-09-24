package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresInstrumentScoringRepository struct {
	db *sql.DB
}

func NewInstrumentScoringRepository(db *sql.DB) domain.InstrumentScoringRepository {
	return &postgresInstrumentScoringRepository{db: db}
}

func (r *postgresInstrumentScoringRepository) PublishVersion(ctx context.Context, instrumentID uuid.UUID, items []domain.InstrumentItem, subscales []domain.InstrumentSubscale, rules []domain.ScoringRule, bands []domain.ScoreBand) (*domain.InstrumentCatalog, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var version int
	err = tx.QueryRowContext(ctx, `
		UPDATE instruments SET version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING version
	`, instrumentID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("instrument %s: %w", instrumentID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("bump instrument version: %w", err)
	}

	for i := range items {
		items[i].ID = uuid.New()
		items[i].InstrumentID = instrumentID
		items[i].InstrumentVersion = version
		_, err = tx.ExecContext(ctx, `
			INSERT INTO instrument_items (id, instrument_id, instrument_version, question_key, subscale_key, order_index, item_type, reverse_scored, weight, value_map, is_scored, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		`, items[i].ID, instrumentID, version, items[i].QuestionKey, items[i].SubscaleKey, items[i].OrderIndex, items[i].ItemType, items[i].ReverseScored, items[i].Weight, defaultJSON(items[i].ValueMap), items[i].IsScored)
		if err != nil {
			return nil, fmt.Errorf("insert instrument item %s: %w", items[i].QuestionKey, err)
		}
	}

	for i := range subscales {
		subscales[i].ID = uuid.New()
		subscales[i].InstrumentID = instrumentID
		subscales[i].InstrumentVersion = version
		_, err = tx.ExecContext(ctx, `
			INSERT INTO instrument_subscales (id, instrument_id, instrument_version, key, name, description, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())
		`, subscales[i].ID, instrumentID, version, subscales[i].Key, subscales[i].Name, subscales[i].Description)
		if err != nil {
			return nil, fmt.Errorf("insert subscale %s: %w", subscales[i].Key, err)
		}
	}

	for i := range rules {
		rules[i].ID = uuid.New()
		rules[i].InstrumentID = instrumentID
		rules[i].InstrumentVersion = version
		_, err = tx.ExecContext(ctx, `
			INSERT INTO instrument_scoring_rules (id, instrument_id, instrument_version, subscale_key, algorithm, min_items_required, missing_strategy, config, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
		`, rules[i].ID, instrumentID, version, rules[i].SubscaleKey, rules[i].Algorithm, rules[i].MinItemsRequired, rules[i].MissingStrategy, defaultJSON(rules[i].Config))
		if err != nil {
			return nil, fmt.Errorf("insert scoring rule: %w", err)
		}
	}

	for i := range bands {
		bands[i].ID = uuid.New()
		bands[i].InstrumentID = instrumentID
		bands[i].InstrumentVersion = version
		_, err = tx.ExecContext(ctx, `
			INSERT INTO instrument_score_bands (id, instrument_id, instrument_version, subscale_key, min_score, max_score, label, description, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		`, bands[i].ID, instrumentID, version, bands[i].SubscaleKey, bands[i].MinScore, bands[i].MaxScore, bands[i].Label, bands[i].Description)
		if err != nil {
			return nil, fmt.Errorf("insert score band %s: %w", bands[i].Label, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit publish version: %w", err)
	}

	return &domain.InstrumentCatalog{InstrumentID: instrumentID, Version: version, Items: items, Subscales: subscales, Rules: rules, Bands: bands}, nil
}

func defaultJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return []byte(raw)
}

func (r *postgresInstrumentScoringRepository) GetLatestCatalog(ctx context.Context, instrumentID uuid.UUID) (*domain.InstrumentCatalog, error) {
	var version int
	err := r.db.QueryRowContext(ctx, `SELECT version FROM instruments WHERE id = $1`, instrumentID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("instrument %s: %w", instrumentID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query instrument version: %w", err)
	}
	return r.GetCatalogByVersion(ctx, instrumentID, version)
}

func (r *postgresInstrumentScoringRepository) GetCatalogByVersion(ctx context.Context, instrumentID uuid.UUID, version int) (*domain.InstrumentCatalog, error) {
	items, err := r.listItems(ctx, instrumentID, version)
	if err != nil {
		return nil, err
	}
	subscales, err := r.listSubscales(ctx, instrumentID, version)
	if err != nil {
		return nil, err
	}
	rules, err := r.listRules(ctx, instrumentID, version)
	if err != nil {
		return nil, err
	}
	bands, err := r.listBands(ctx, instrumentID, version)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 && len(rules) == 0 {
		return nil, fmt.Errorf("catalog for instrument %s version %d: %w", instrumentID, version, domain.ErrNotFound)
	}
	return &domain.InstrumentCatalog{InstrumentID: instrumentID, Version: version, Items: items, Subscales: subscales, Rules: rules, Bands: bands}, nil
}

func (r *postgresInstrumentScoringRepository) listItems(ctx context.Context, instrumentID uuid.UUID, version int) ([]domain.InstrumentItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, instrument_id, instrument_version, question_key, subscale_key, order_index, item_type, reverse_scored, weight, value_map, is_scored, created_at, updated_at
		FROM instrument_items WHERE instrument_id = $1 AND instrument_version = $2 ORDER BY order_index
	`, instrumentID, version)
	if err != nil {
		return nil, fmt.Errorf("list instrument items: %w", err)
	}
	defer rows.Close()
	items := make([]domain.InstrumentItem, 0)
	for rows.Next() {
		var it domain.InstrumentItem
		var subscaleKey sql.NullString
		var valueMap []byte
		if err := rows.Scan(&it.ID, &it.InstrumentID, &it.InstrumentVersion, &it.QuestionKey, &subscaleKey, &it.OrderIndex, &it.ItemType, &it.ReverseScored, &it.Weight, &valueMap, &it.IsScored, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan instrument item: %w", err)
		}
		if subscaleKey.Valid {
			it.SubscaleKey = &subscaleKey.String
		}
		it.ValueMap = json.RawMessage(valueMap)
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *postgresInstrumentScoringRepository) listSubscales(ctx context.Context, instrumentID uuid.UUID, version int) ([]domain.InstrumentSubscale, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, instrument_id, instrument_version, key, name, description, created_at, updated_at
		FROM instrument_subscales WHERE instrument_id = $1 AND instrument_version = $2
	`, instrumentID, version)
	if err != nil {
		return nil, fmt.Errorf("list subscales: %w", err)
	}
	defer rows.Close()
	subscales := make([]domain.InstrumentSubscale, 0)
	for rows.Next() {
		var s domain.InstrumentSubscale
		var description sql.NullString
		if err := rows.Scan(&s.ID, &s.InstrumentID, &s.InstrumentVersion, &s.Key, &s.Name, &description, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan subscale: %w", err)
		}
		if description.Valid {
			s.Description = &description.String
		}
		subscales = append(subscales, s)
	}
	return subscales, rows.Err()
}

func (r *postgresInstrumentScoringRepository) listRules(ctx context.Context, instrumentID uuid.UUID, version int) ([]domain.ScoringRule, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, instrument_id, instrument_version, subscale_key, algorithm, min_items_required, missing_strategy, config, created_at, updated_at
		FROM instrument_scoring_rules WHERE instrument_id = $1 AND instrument_version = $2
	`, instrumentID, version)
	if err != nil {
		return nil, fmt.Errorf("list scoring rules: %w", err)
	}
	defer rows.Close()
	rules := make([]domain.ScoringRule, 0)
	for rows.Next() {
		var rule domain.ScoringRule
		var subscaleKey sql.NullString
		var minItems sql.NullInt64
		var config []byte
		if err := rows.Scan(&rule.ID, &rule.InstrumentID, &rule.InstrumentVersion, &subscaleKey, &rule.Algorithm, &minItems, &rule.MissingStrategy, &config, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan scoring rule: %w", err)
		}
		if subscaleKey.Valid {
			rule.SubscaleKey = &subscaleKey.String
		}
		if minItems.Valid {
			v := int(minItems.Int64)
			rule.MinItemsRequired = &v
		}
		rule.Config = json.RawMessage(config)
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *postgresInstrumentScoringRepository) listBands(ctx context.Context, instrumentID uuid.UUID, version int) ([]domain.ScoreBand, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, instrument_id, instrument_version, subscale_key, min_score, max_score, label, description, created_at
		FROM instrument_score_bands WHERE instrument_id = $1 AND instrument_version = $2
	`, instrumentID, version)
	if err != nil {
		return nil, fmt.Errorf("list score bands: %w", err)
	}
	defer rows.Close()
	bands := make([]domain.ScoreBand, 0)
	for rows.Next() {
		var b domain.ScoreBand
		var subscaleKey sql.NullString
		var minScore, maxScore sql.NullFloat64
		var description sql.NullString
		if err := rows.Scan(&b.ID, &b.InstrumentID, &b.InstrumentVersion, &subscaleKey, &minScore, &maxScore, &b.Label, &description, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan score band: %w", err)
		}
		if subscaleKey.Valid {
			b.SubscaleKey = &subscaleKey.String
		}
		if minScore.Valid {
			b.MinScore = &minScore.Float64
		}
		if maxScore.Valid {
			b.MaxScore = &maxScore.Float64
		}
		if description.Valid {
			b.Description = &description.String
		}
		bands = append(bands, b)
	}
	return bands, rows.Err()
}

func (r *postgresInstrumentScoringRepository) ListVersions(ctx context.Context, instrumentID uuid.UUID) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT instrument_version FROM instrument_scoring_rules WHERE instrument_id = $1 ORDER BY instrument_version DESC
	`, instrumentID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	versions := make([]int, 0)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}
