package models

import (
	"time"
)

// Skin represents the core information about a CS2 skin
type Skin struct {
	ID             string    `json:"id" db:"id"`
	MarketHashName string    `json:"market_hash_name" db:"market_hash_name"`
	Category       string    `json:"category" db:"category"`         // e.g. Rifle, Knife, Pistol
	SubCategory    string    `json:"sub_category" db:"sub_category"` // e.g. AK-47, Karambit, USP-S
	SkinName       string    `json:"skin_name" db:"skin_name"`       // e.g. Asiimov, Fade, Doppler
	IsStatTrak     bool      `json:"is_stattrak" db:"is_stattrak"`
	Quality        string    `json:"quality" db:"quality"`     // Factory New, Minimal Wear, etc.
	MinFloat       float64   `json:"min_float" db:"min_float"` // Minimum possible float value
	MaxFloat       float64   `json:"max_float" db:"max_float"` // Maximum possible float value
	IconURL        string    `json:"icon_url" db:"icon_url"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// GetWearCategory returns the wear category based on a float value
func GetWearCategory(floatValue float64) string {
	switch {
	case floatValue >= 0 && floatValue < 0.07:
		return "Factory New"
	case floatValue >= 0.07 && floatValue < 0.15:
		return "Minimal Wear"
	case floatValue >= 0.15 && floatValue < 0.38:
		return "Field-Tested"
	case floatValue >= 0.38 && floatValue < 0.45:
		return "Well-Worn"
	case floatValue >= 0.45 && floatValue <= 1.0:
		return "Battle-Scarred"
	default:
		return "Unknown"
	}
}
