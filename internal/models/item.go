package models

import (
	"time"
)

// Item represents a specific instance of a skin in a marketplace
type Item struct {
	ID            string    `json:"id" db:"id"`
	SkinID        string    `json:"skin_id" db:"skin_id"`
	MarketplaceID string    `json:"marketplace_id" db:"marketplace_id"`
	Float         float64   `json:"float" db:"float"`
	Stickers      []string  `json:"stickers" db:"stickers"`
	Price         float64   `json:"price" db:"price"`                     // Price in marketplace currency
	PriceFailed   float64   `json:"price_failed" db:"price_failed"`       // Original price before discount
	PriceUSD      float64   `json:"price_usd" db:"price_usd"`             // Converted price in USD
	SteamPriceUSD float64   `json:"steam_price_usd" db:"steam_price_usd"` // Steam market price in USD
	Tradeable     string    `json:"tradeable" db:"tradeable"`
	IsFastSell    bool      `json:"is_fast_sell" db:"is_fast_sell"`
	MarketItemID  string    `json:"market_item_id" db:"market_item_id"` // Original ID in the marketplace
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CSGOSkinItem represents the structure returned by csgoskin.ir API
type CSGOSkinItem struct {
	ItemID string `json:"itemid"`
	Name   struct {
		Category string `json:"category"`
		SkinName string `json:"skinname"`
	} `json:"name"`
	IsStatTrack    int      `json:"is_stattrack"`
	Quality        string   `json:"quality"`
	Price          string   `json:"price"`
	PriceFailed    string   `json:"price_faild"`
	Float          string   `json:"float"`
	MarketHashName string   `json:"market_hash_name"`
	Tradeable      string   `json:"tradeable"`
	PriceSteam     string   `json:"price_steam"`
	IconMedium     string   `json:"icon_medium"`
	Stickers       []string `json:"stickers"`
}
