package models

import (
	"time"
)

// Marketplace represents a skin marketplace
type Marketplace struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	URL       string    `json:"url" db:"url"`
	Currency  string    `json:"currency" db:"currency"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MarketplaceFee represents the fees associated with a marketplace
type MarketplaceFee struct {
	ID                 string  `json:"id" db:"id"`
	MarketplaceID      string  `json:"marketplace_id" db:"marketplace_id"`
	ListingFeePercent  float64 `json:"listing_fee_percent" db:"listing_fee_percent"`
	SaleFeePercent     float64 `json:"sale_fee_percent" db:"sale_fee_percent"`
	FastSellFeePercent float64 `json:"fast_sell_fee_percent" db:"fast_sell_fee_percent"`
}