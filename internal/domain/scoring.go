package domain

import (
	"encoding/json"
	"fmt"
)

// Engine is the default, pure-function implementation of domain.ScoringEngine.
// It has no side effects and no DB access, so it can be unit tested directly
// against fixtures without a database.
type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

// answerEnvelope is the ASSUMED shape of answer_records.value for scorable
// items: {"value": <string|number>}. This is the one place to change if the
// actual client payload differs — nothing else in the engine depends on it.
type answerEnvelope struct {
	Value json.RawMessage `json:"value"`
}

func (e *Engine) Compute(items []InstrumentItem, rules []ScoringRule, bands []ScoreBand, answers []AnswerRecord) ([]InstrumentScore, error) {
	answersByKey := make(map[string]AnswerRecord, len(answers))
	for _, a := range answers {
		answersByKey[a.QuestionKey] = a
	}

	itemsBySubscale := make(map[string][]InstrumentItem)
	var totalItems []InstrumentItem
	for _, it := range items {
		if !it.IsScored {
			continue
		}
		totalItems = append(totalItems, it)
		key := ""
		if it.SubscaleKey != nil {
			key = *it.SubscaleKey
		}
		itemsBySubscale[key] = append(itemsBySubscale[key], it)
	}

	bandsBySubscale := make(map[string][]ScoreBand)
	for _, b := range bands {
		key := ""
		if b.SubscaleKey != nil {
			key = *b.SubscaleKey
		}
		bandsBySubscale[key] = append(bandsBySubscale[key], b)
	}

	scores := make([]InstrumentScore, 0, len(rules))
	for _, rule := range rules {
		key := ""
		if rule.SubscaleKey != nil {
			key = *rule.SubscaleKey
		}
		scopedItems := totalItems
		if key != "" {
			scopedItems = itemsBySubscale[key]
		}

		score, err := computeOne(rule, scopedItems, answersByKey)
		if err != nil {
			return nil, fmt.Errorf("compute rule (subscale=%q): %w", key, err)
		}

		if score.RawScore != nil {
			for _, b := range bandsBySubscale[key] {
				if withinBand(*score.RawScore, b) {
					bandID, label := b.ID, b.Label
					score.BandID, score.BandLabel = &bandID, &label
					break
				}
			}
		}
		scores = append(scores, score)
	}
	return scores, nil
}

func computeOne(rule ScoringRule, items []InstrumentItem, answersByKey map[string]AnswerRecord) (InstrumentScore, error) {
	score := InstrumentScore{SubscaleKey: rule.SubscaleKey, ItemsExpected: len(items)}

	var sum, weightSum float64
	answered := 0
	for _, item := range items {
		answer, ok := answersByKey[item.QuestionKey]
		if !ok {
			continue
		}
		value, ok, err := extractItemScore(item, answer)
		if err != nil {
			return score, fmt.Errorf("item %q: %w", item.QuestionKey, err)
		}
		if !ok {
			continue
		}
		if item.ReverseScored {
			value = reverse(value, item)
		}
		weight := item.Weight
		if weight == 0 {
			weight = 1
		}
		sum += value * weight
		weightSum += weight
		answered++
	}
	score.ItemsAnswered = answered

	if rule.MinItemsRequired != nil && answered < *rule.MinItemsRequired {
		return score, nil // not enough data: raw/scaled stay nil
	}
	if answered == 0 {
		if rule.MissingStrategy == MissingStrategyZero {
			zero := 0.0
			score.RawScore = &zero
		}
		return score, nil
	}

	var raw float64
	switch rule.Algorithm {
	case ScoringAlgorithmSum, ScoringAlgorithmWeightedSum:
		raw = applyMissingStrategy(sum, answered, len(items), rule.MissingStrategy)
	case ScoringAlgorithmAverage:
		if weightSum == 0 {
			return score, nil
		}
		raw = sum / weightSum
	case ScoringAlgorithmLookupTable, ScoringAlgorithmCustom:
		// These need a table/expression carried in rule.Config, which is
		// instrument-specific and not generically computable here. We leave
		// the plain sum as raw so a caller can post-process using
		// ComputationVersion as a flag that this needs special handling.
		raw = sum
	default:
		return score, fmt.Errorf("unknown algorithm %q", rule.Algorithm)
	}
	score.RawScore = &raw
	return score, nil
}

// applyMissingStrategy scales a sum computed from a subset of items up to
// what it would be if every expected item had been answered ("prorate").
// For "zero" and "null_if_missing" the sum is left as-is (unanswered items
// already contribute 0, since they were simply skipped in the loop above).
func applyMissingStrategy(sum float64, answered, expected int, strategy MissingStrategy) float64 {
	if strategy != MissingStrategyProrate || answered == 0 || answered == expected {
		return sum
	}
	return sum * (float64(expected) / float64(answered))
}

func reverse(value float64, item InstrumentItem) float64 {
	max := maxValueMap(item.ValueMap)
	if max == 0 {
		return value
	}
	return max - value
}

func maxValueMap(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var m map[string]float64
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0
	}
	max := 0.0
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}

func withinBand(score float64, band ScoreBand) bool {
	if band.MinScore != nil && score < *band.MinScore {
		return false
	}
	if band.MaxScore != nil && score > *band.MaxScore {
		return false
	}
	return true
}

// extractItemScore reads answer.Value and returns the numeric score it
// represents for this item, using item.ValueMap when the answer is a label
// rather than a raw number. Returns ok=false when the answer carries no
// scorable value (e.g. an empty text response for an unscored item).
func extractItemScore(item InstrumentItem, answer AnswerRecord) (float64, bool, error) {
	var envelope answerEnvelope
	if err := json.Unmarshal(answer.Value, &envelope); err != nil || len(envelope.Value) == 0 {
		return 0, false, nil
	}

	var asNumber float64
	if err := json.Unmarshal(envelope.Value, &asNumber); err == nil {
		return asNumber, true, nil
	}

	var asString string
	if err := json.Unmarshal(envelope.Value, &asString); err == nil {
		if len(item.ValueMap) == 0 {
			return 0, false, fmt.Errorf("answer is a label %q but item has no value_map", asString)
		}
		var valueMap map[string]float64
		if err := json.Unmarshal(item.ValueMap, &valueMap); err != nil {
			return 0, false, fmt.Errorf("invalid value_map: %w", err)
		}
		mapped, found := valueMap[asString]
		if !found {
			return 0, false, fmt.Errorf("label %q not present in value_map", asString)
		}
		return mapped, true, nil
	}

	return 0, false, nil
}
