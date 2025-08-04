package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mswatii/cs2-arbitrage/internal/models"
	"os"
)

type Database struct {
	pool *pgxpool.Pool
}

// NewDatabase creates a new database connection
func NewDatabase() (*Database, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	return &Database{pool: pool}, nil
}

// Close closes the database connection
func (db *Database) Close() {
	db.pool.Close()
}

// CreateTables creates the necessary tables if they don't exist
func (db *Database) CreateTables() error {
	// Create marketplaces table
	_, err := db.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS marketplaces (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			url VARCHAR(255) NOT NULL,
			currency VARCHAR(10) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating marketplaces table: %v", err)
	}

	// Create marketplace_fees table
	_, err = db.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS marketplace_fees (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			marketplace_id UUID REFERENCES marketplaces(id),
			listing_fee_percent DECIMAL(5,2) NOT NULL,
			sale_fee_percent DECIMAL(5,2) NOT NULL,
			fast_sell_fee_percent DECIMAL(5,2) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating marketplace_fees table: %v", err)
	}

	// Create skins table
	_, err = db.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS skins (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			market_hash_name VARCHAR(255) NOT NULL UNIQUE,
			category VARCHAR(255) NOT NULL,
			sub_category VARCHAR(255) NOT NULL,
			skin_name VARCHAR(255) NOT NULL,
			is_stattrak BOOLEAN NOT NULL DEFAULT false,
			quality VARCHAR(50) NOT NULL,
			min_float DECIMAL(18,16),
			max_float DECIMAL(18,16),
			icon_url TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating skins table: %v", err)
	}

	// Create items table (specific instances in marketplaces)
	_, err = db.pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			skin_id UUID REFERENCES skins(id),
			marketplace_id UUID REFERENCES marketplaces(id),
			float DECIMAL(18,16),
			stickers TEXT[],
			price DECIMAL(15,2) NOT NULL,
			price_failed DECIMAL(15,2),
			price_usd DECIMAL(15,2),
			steam_price_usd DECIMAL(15,2),
			tradeable VARCHAR(50),
			is_fast_sell BOOLEAN NOT NULL DEFAULT false,
			market_item_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(marketplace_id, market_item_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating items table: %v", err)
	}

	return nil
}

// GetSkinByMarketHashName retrieves a skin by its market hash name
func (db *Database) GetSkinByMarketHashName(marketHashName string) (*models.Skin, error) {
	skin := &models.Skin{}
	err := db.pool.QueryRow(context.Background(), `
		SELECT id, market_hash_name, category, sub_category, skin_name, 
		       is_stattrak, quality, min_float, max_float, icon_url
		FROM skins WHERE market_hash_name = $1
	`, marketHashName).Scan(
		&skin.ID, &skin.MarketHashName, &skin.Category, &skin.SubCategory, &skin.SkinName,
		&skin.IsStatTrak, &skin.Quality, &skin.MinFloat, &skin.MaxFloat, &skin.IconURL,
	)
	if err != nil {
		return nil, fmt.Errorf("skin not found: %v", err)
	}
	return skin, nil
}

// InsertSkin inserts a skin into the database
func (db *Database) InsertSkin(skin *models.Skin) (string, error) {
	var id string
	err := db.pool.QueryRow(context.Background(), `
		INSERT INTO skins (
			market_hash_name, category, sub_category, skin_name, is_stattrak,
			quality, min_float, max_float, icon_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (market_hash_name) 
		DO UPDATE SET 
			category = $2,
			sub_category = $3,
			skin_name = $4,
			is_stattrak = $5,
			quality = $6,
			min_float = $7,
			max_float = $8,
			icon_url = $9,
			updated_at = NOW()
		RETURNING id
	`,
		skin.MarketHashName, skin.Category, skin.SubCategory, skin.SkinName, skin.IsStatTrak,
		skin.Quality, skin.MinFloat, skin.MaxFloat, skin.IconURL,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("error inserting skin: %v", err)
	}

	return id, nil
}

// InsertItem inserts an item into the database
func (db *Database) InsertItem(item *models.Item) (string, error) {
	var id string
	err := db.pool.QueryRow(context.Background(), `
		INSERT INTO items (
			skin_id, marketplace_id, float, stickers, price, price_failed,
			price_usd, steam_price_usd, tradeable, is_fast_sell, market_item_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (marketplace_id, market_item_id) 
		DO UPDATE SET 
			skin_id = $1,
			float = $3,
			stickers = $4,
			price = $5,
			price_failed = $6,
			price_usd = $7,
			steam_price_usd = $8,
			tradeable = $9,
			is_fast_sell = $10,
			updated_at = NOW()
		RETURNING id
	`,
		item.SkinID, item.MarketplaceID, item.Float, item.Stickers, item.Price, item.PriceFailed,
		item.PriceUSD, item.SteamPriceUSD, item.Tradeable, item.IsFastSell, item.MarketItemID,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("error inserting item: %v", err)
	}

	return id, nil
}

// InsertMarketplace inserts a marketplace into the database
func (db *Database) InsertMarketplace(marketplace *models.Marketplace) (string, error) {
	var id string
	err := db.pool.QueryRow(context.Background(), `
		INSERT INTO marketplaces (
			name, url, currency
		) VALUES ($1, $2, $3)
		ON CONFLICT (name) 
		DO UPDATE SET 
			url = $2,
			currency = $3,
			updated_at = NOW()
		RETURNING id
	`,
		marketplace.Name, marketplace.URL, marketplace.Currency,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("error inserting marketplace: %v", err)
	}

	return id, nil
}

// ExecuteQuery executes a SQL query and returns the results
func (db *Database) ExecuteQuery(query string, args ...interface{}) ([]struct {
	MarketHashName string
	BuyPriceUSD    float64
	SellPriceUSD   float64
	ProfitUSD      float64
	ProfitPercent  float64
	Marketplace    string
	Float          float64
	Quality        string
	IconURL        string
	Category       string
	IsStatTrak     bool
	Stickers       []string
}, error) {
	rows, err := db.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var results []struct {
		MarketHashName string
		BuyPriceUSD    float64
		SellPriceUSD   float64
		ProfitUSD      float64
		ProfitPercent  float64
		Marketplace    string
		Float          float64
		Quality        string
		IconURL        string
		Category       string
		IsStatTrak     bool
		Stickers       []string
	}

	for rows.Next() {
		var result struct {
			MarketHashName string
			BuyPriceUSD    float64
			SellPriceUSD   float64
			ProfitUSD      float64
			ProfitPercent  float64
			Marketplace    string
			Float          float64
			Quality        string
			IconURL        string
			Category       string
			IsStatTrak     bool
			Stickers       []string
		}

		err := rows.Scan(
			&result.MarketHashName,
			&result.BuyPriceUSD,
			&result.SellPriceUSD,
			&result.ProfitUSD,
			&result.ProfitPercent,
			&result.Marketplace,
			&result.Float,
			&result.Quality,
			&result.IconURL,
			&result.Category,
			&result.IsStatTrak,
			&result.Stickers,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return results, nil
}
