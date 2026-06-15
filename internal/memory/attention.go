package memory

import (
	"math"
	"time"
)

// CalculateScore computes the attention score for an entry.
//
//	score = load_freq * 0.3 + recency * 0.3 + starred * 0.2 + ai_weight * 0.2
func (e *Entry) CalculateScore() float64 {
	loadFreq := normalizeLoadCount(e.LoadCount)
	recency := normalizeRecency(e.LastAccess)
	starBoost := 0.0
	if e.Starred {
		starBoost = 1.0
	}
	// ai_weight is stored on the entry; default 0.5 if unset
	aiWeight := e.Score
	if aiWeight <= 0 {
		aiWeight = 0.5
	}

	score := loadFreq*0.3 + recency*0.3 + starBoost*0.2 + aiWeight*0.2
	return math.Max(0, math.Min(1, score))
}

// normalizeLoadCount maps load count to a 0-1 range using a log scale.
// 1 load -> ~0.0, 10 loads -> ~0.5, 100+ loads -> ~1.0
func normalizeLoadCount(count int) float64 {
	if count <= 0 {
		return 0
	}
	// log10(1)=0, log10(10)=1, log10(100)=2 -> divide by 2 for 0-1
	return math.Min(1, math.Log10(float64(count))/2.0)
}

// normalizeRecency returns 1.0 if accessed today, decaying linearly to 0 over 30 days.
func normalizeRecency(lastAccess time.Time) float64 {
	if lastAccess.IsZero() {
		return 0
	}
	now := time.Now()
	days := now.Sub(lastAccess).Hours() / 24.0
	if days < 0 {
		days = 0
	}
	if days >= 30 {
		return 0
	}
	return 1.0 - days/30.0
}

// Star toggles the starred flag and applies an AI-assigned weight.
// If aiWeight is 0, the existing score is preserved.
func (e *Entry) Star(aiWeight float64) {
	e.Starred = !e.Starred
	if aiWeight > 0 {
		e.Score = aiWeight
	}
}
