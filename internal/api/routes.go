package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mswatii/cs2-arbitrage/internal/database"
	"github.com/mswatii/cs2-arbitrage/internal/scraper"
	"github.com/valyala/fasthttp"
)

// Handler represents the API handler
type Handler struct {
	db *database.Database
}

// NewHandler creates a new API handler
func NewHandler(db *database.Database) *Handler {
	return &Handler{
		db: db,
	}
}

// Update your HandleRequest function in routes.go
func (h *Handler) HandleRequest(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	// Handle web routes first
	if path == "/" || path == "/index.html" {
		h.handleIndex(ctx)
		return
	}

	// Handle static files
	if strings.HasPrefix(path, "/static/") {
		h.handleStatic(ctx)
		return
	}

	// Handle API routes
	switch {
	case path == "/api/health":
		h.handleHealth(ctx)
	case path == "/api/refresh":
		h.handleRefresh(ctx)
	case path == "/api/exchange-rate":
		h.handleExchangeRate(ctx)
	case path == "/api/arbitrage":
		h.handleArbitrage(ctx)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
	}
}

// handleHealth handles the health check endpoint
func (h *Handler) handleHealth(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(ctx).Encode(response)
}

// handleRefresh handles the data refresh endpoint
func (h *Handler) handleRefresh(ctx *fasthttp.RequestCtx) {
	// Create a scraper and fetch data
	csgoSkinScraper, err := scraper.NewCSGOSkinScraper(h.db)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf("Failed to initialize scraper: %v", err))
		return
	}

	err = csgoSkinScraper.FetchItems()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf("Failed to fetch items: %v", err))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("Data refreshed successfully")
}

// handleExchangeRate handles the exchange rate endpoint
func (h *Handler) handleExchangeRate(ctx *fasthttp.RequestCtx) {
	usdtToIRR := scraper.GetUSDTtoIRRRate()
	irrToUSD := scraper.GetIRRtoUSDRate()

	response := map[string]interface{}{
		"usdt_to_irr": usdtToIRR,
		"irr_to_usd":  irrToUSD,
		"updated_at":  time.Now().Format(time.RFC3339),
		"note":        "1 USDT = X IRR, 1 IRR = Y USD",
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// handleArbitrage handles the arbitrage opportunities endpoint
func (h *Handler) handleArbitrage(ctx *fasthttp.RequestCtx) {
	// Parse min profit percentage from query params (default 10%)
	minProfitStr := string(ctx.QueryArgs().Peek("min_profit"))
	minProfit := 10.0 // default
	if minProfitStr != "" {
		if parsedProfit, err := json.Number(minProfitStr).Float64(); err == nil {
			minProfit = parsedProfit
		}
	}

	// Use the FindArbitrageOpportunities function directly
	// since we can't access db.pool directly
	opportunities, err := findArbitrageOpportunities(h.db, minProfit)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf("Failed to find arbitrage opportunities: %v", err))
		return
	}

	response := map[string]interface{}{
		"opportunities":      opportunities,
		"count":              len(opportunities),
		"min_profit_percent": minProfit,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// ArbitrageOpportunity represents a potential arbitrage opportunity
type ArbitrageOpportunity struct {
	MarketHashName string   `json:"market_hash_name"`
	BuyPriceUSD    float64  `json:"buy_price_usd"`
	SellPriceUSD   float64  `json:"sell_price_usd"`
	ProfitUSD      float64  `json:"profit_usd"`
	ProfitPercent  float64  `json:"profit_percent"`
	Marketplace    string   `json:"marketplace"`
	Float          float64  `json:"float"`
	Quality        string   `json:"quality"`
	IconURL        string   `json:"icon_url"`
	Category       string   `json:"category"`
	IsStatTrak     bool     `json:"is_stattrak"`
	Stickers       []string `json:"stickers"`
}

// findArbitrageOpportunities finds arbitrage opportunities using the database struct
func findArbitrageOpportunities(db *database.Database, minProfitPercent float64) ([]ArbitrageOpportunity, error) {
	// Update the query to include icon_url from skins table
	query := `
        SELECT 
            s.market_hash_name, 
            i.price_usd as buy_price_usd, 
            i.steam_price_usd as sell_price_usd,
            (i.steam_price_usd - i.price_usd) AS profit_usd,
            (i.steam_price_usd - i.price_usd) / i.price_usd * 100 AS profit_percent,
            m.name AS marketplace,
            i.float,
            s.quality,
            s.icon_url, 
            s.category,
            s.is_stattrak,
            i.stickers
        FROM 
            items i
        JOIN 
            skins s ON i.skin_id = s.id
        JOIN 
            marketplaces m ON i.marketplace_id = m.id
        WHERE 
            i.steam_price_usd > 0
            AND i.price_usd > 0
            AND (i.steam_price_usd - i.price_usd) / i.price_usd * 100 >= $1
        ORDER BY 
            profit_percent DESC
    `

	// We'll need to add a method to your database struct
	// But for now we can add this helper method here

	// Let's use the existing QueryRow method that should be available
	rows, err := db.ExecuteQuery(query, minProfitPercent)
	if err != nil {
		return nil, fmt.Errorf("error querying arbitrage opportunities: %v", err)
	}

	var opportunities []ArbitrageOpportunity

	for _, row := range rows {
		opp := ArbitrageOpportunity{
			MarketHashName: row.MarketHashName,
			BuyPriceUSD:    row.BuyPriceUSD,
			SellPriceUSD:   row.SellPriceUSD,
			ProfitUSD:      row.ProfitUSD,
			ProfitPercent:  row.ProfitPercent,
			Marketplace:    row.Marketplace,
			Float:          row.Float,
			Quality:        row.Quality,
			IconURL:        row.IconURL,
			Category:       row.Category,
			IsStatTrak:     row.IsStatTrak,
			Stickers:       row.Stickers,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}
