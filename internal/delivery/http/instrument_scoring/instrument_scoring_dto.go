package instrument_scoring

import (
	"encoding/json"

	"indagio-api/internal/domain"
)

type ItemRequest struct {
	QuestionKey   string          `json:"question_key" binding:"required,max=120"`
	SubscaleKey   *string         `json:"subscale_key"`
	OrderIndex    int             `json:"order_index"`
	ItemType      string          `json:"item_type" binding:"required"`
	ReverseScored bool            `json:"reverse_scored"`
	Weight        float64         `json:"weight"`
	ValueMap      json.RawMessage `json:"value_map" swaggertype:"object"`
	IsScored      *bool           `json:"is_scored"`
}

type SubscaleRequest struct {
	Key         string  `json:"key" binding:"required,max=64"`
	Name        string  `json:"name" binding:"required,max=160"`
	Description *string `json:"description"`
}

type ScoringRuleRequest struct {
	SubscaleKey      *string         `json:"subscale_key"`
	Algorithm        string          `json:"algorithm" binding:"required"`
	MinItemsRequired *int            `json:"min_items_required"`
	MissingStrategy  string          `json:"missing_strategy" binding:"required"`
	Config           json.RawMessage `json:"config" swaggertype:"object"`
}

type ScoreBandRequest struct {
	SubscaleKey *string  `json:"subscale_key"`
	MinScore    *float64 `json:"min_score"`
	MaxScore    *float64 `json:"max_score"`
	Label       string   `json:"label" binding:"required,max=120"`
	Description *string  `json:"description"`
}

type PublishVersionRequest struct {
	Items     []ItemRequest        `json:"items" binding:"required,min=1,dive"`
	Subscales []SubscaleRequest    `json:"subscales" binding:"dive"`
	Rules     []ScoringRuleRequest `json:"rules" binding:"required,min=1,dive"`
	Bands     []ScoreBandRequest   `json:"bands" binding:"dive"`
}

func (req PublishVersionRequest) ToDomain() ([]domain.InstrumentItem, []domain.InstrumentSubscale, []domain.ScoringRule, []domain.ScoreBand) {
	items := make([]domain.InstrumentItem, 0, len(req.Items))
	for _, it := range req.Items {
		isScored := true
		if it.IsScored != nil {
			isScored = *it.IsScored
		}
		items = append(items, domain.InstrumentItem{
			QuestionKey: it.QuestionKey, SubscaleKey: it.SubscaleKey, OrderIndex: it.OrderIndex,
			ItemType: it.ItemType, ReverseScored: it.ReverseScored, Weight: it.Weight,
			ValueMap: it.ValueMap, IsScored: isScored,
		})
	}
	subscales := make([]domain.InstrumentSubscale, 0, len(req.Subscales))
	for _, s := range req.Subscales {
		subscales = append(subscales, domain.InstrumentSubscale{Key: s.Key, Name: s.Name, Description: s.Description})
	}
	rules := make([]domain.ScoringRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		rules = append(rules, domain.ScoringRule{
			SubscaleKey: r.SubscaleKey, Algorithm: domain.ScoringAlgorithm(r.Algorithm),
			MinItemsRequired: r.MinItemsRequired, MissingStrategy: domain.MissingStrategy(r.MissingStrategy), Config: r.Config,
		})
	}
	bands := make([]domain.ScoreBand, 0, len(req.Bands))
	for _, b := range req.Bands {
		bands = append(bands, domain.ScoreBand{
			SubscaleKey: b.SubscaleKey, MinScore: b.MinScore, MaxScore: b.MaxScore,
			Label: b.Label, Description: b.Description,
		})
	}
	return items, subscales, rules, bands
}

type ItemResponse struct {
	ID            string          `json:"id"`
	QuestionKey   string          `json:"question_key"`
	SubscaleKey   *string         `json:"subscale_key"`
	OrderIndex    int             `json:"order_index"`
	ItemType      string          `json:"item_type"`
	ReverseScored bool            `json:"reverse_scored"`
	Weight        float64         `json:"weight"`
	ValueMap      json.RawMessage `json:"value_map" swaggertype:"object"`
	IsScored      bool            `json:"is_scored"`
}

type SubscaleResponse struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type ScoringRuleResponse struct {
	SubscaleKey      *string         `json:"subscale_key"`
	Algorithm        string          `json:"algorithm"`
	MinItemsRequired *int            `json:"min_items_required"`
	MissingStrategy  string          `json:"missing_strategy"`
	Config           json.RawMessage `json:"config" swaggertype:"object"`
}

type ScoreBandResponse struct {
	SubscaleKey *string  `json:"subscale_key"`
	MinScore    *float64 `json:"min_score"`
	MaxScore    *float64 `json:"max_score"`
	Label       string   `json:"label"`
	Description *string  `json:"description"`
}

type CatalogResponse struct {
	InstrumentID string                `json:"instrument_id"`
	Version      int                   `json:"version"`
	Items        []ItemResponse        `json:"items"`
	Subscales    []SubscaleResponse    `json:"subscales"`
	Rules        []ScoringRuleResponse `json:"rules"`
	Bands        []ScoreBandResponse   `json:"bands"`
}

func ToCatalogResponse(c *domain.InstrumentCatalog) CatalogResponse {
	resp := CatalogResponse{InstrumentID: c.InstrumentID.String(), Version: c.Version}
	for _, it := range c.Items {
		resp.Items = append(resp.Items, ItemResponse{
			ID: it.ID.String(), QuestionKey: it.QuestionKey, SubscaleKey: it.SubscaleKey,
			OrderIndex: it.OrderIndex, ItemType: it.ItemType, ReverseScored: it.ReverseScored,
			Weight: it.Weight, ValueMap: it.ValueMap, IsScored: it.IsScored,
		})
	}
	for _, s := range c.Subscales {
		resp.Subscales = append(resp.Subscales, SubscaleResponse{Key: s.Key, Name: s.Name, Description: s.Description})
	}
	for _, r := range c.Rules {
		resp.Rules = append(resp.Rules, ScoringRuleResponse{
			SubscaleKey: r.SubscaleKey, Algorithm: string(r.Algorithm),
			MinItemsRequired: r.MinItemsRequired, MissingStrategy: string(r.MissingStrategy), Config: r.Config,
		})
	}
	for _, b := range c.Bands {
		resp.Bands = append(resp.Bands, ScoreBandResponse{
			SubscaleKey: b.SubscaleKey, MinScore: b.MinScore, MaxScore: b.MaxScore,
			Label: b.Label, Description: b.Description,
		})
	}
	return resp
}
