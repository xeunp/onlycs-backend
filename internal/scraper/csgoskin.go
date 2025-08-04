package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mswatii/cs2-arbitrage/internal/database"
	"github.com/mswatii/cs2-arbitrage/internal/models"
	"github.com/valyala/fasthttp"
)

const (
	CSGOSkinURL             = "https://csgoskin.ir/ajax.php?action=loaditem"
	CSGOSkinMarketplaceName = "CSGOSkin.ir"
	CSGOSkinCurrency        = "IRR" // Iranian Rial
	RequestDelayMs          = 500   // Delay between paginated requests (milliseconds)
	MaxItemsToFetch         = 50000 // Safety limit to avoid infinite loops
)

// CSGOSkinScraper handles scraping data from csgoskin.ir
type CSGOSkinScraper struct {
	db            *database.Database
	marketplaceID string
}

// NewCSGOSkinScraper creates a new scraper for csgoskin.ir
func NewCSGOSkinScraper(db *database.Database) (*CSGOSkinScraper, error) {
	// Insert or get marketplace ID
	marketplace := &models.Marketplace{
		Name:     CSGOSkinMarketplaceName,
		URL:      "https://csgoskin.ir",
		Currency: CSGOSkinCurrency,
	}

	marketplaceID, err := db.InsertMarketplace(marketplace)
	if err != nil {
		return nil, fmt.Errorf("failed to insert marketplace: %v", err)
	}

	return &CSGOSkinScraper{
		db:            db,
		marketplaceID: marketplaceID,
	}, nil
}

// FetchItems fetches all items from csgoskin.ir using pagination
func (s *CSGOSkinScraper) FetchItems() error {
	var lastItemID string = "0" // Start with 0 for the first page
	var totalItemsProcessed int = 0
	var totalPages int = 0

	for {
		totalPages++
		log.Printf("Fetching page %d (last item ID: %s)...", totalPages, lastItemID)

		// Fetch items for the current page
		csgoItems, newLastItemID, err := s.fetchItemsPage(lastItemID)
		if err != nil {
			return fmt.Errorf("error fetching page %d: %v", totalPages, err)
		}

		itemCount := len(csgoItems)
		log.Printf("Fetched %d items from page %d", itemCount, totalPages)

		// Process items from this page
		for _, csgoItem := range csgoItems {
			if err := s.processItem(csgoItem); err != nil {
				log.Printf("Error processing item %s: %v", csgoItem.MarketHashName, err)
				continue
			}
			totalItemsProcessed++
		}

		// Check if we've reached the end (no more items or same last item ID)
		if itemCount == 0 || newLastItemID == lastItemID {
			log.Printf("Reached the end of pagination. Total items processed: %d", totalItemsProcessed)
			break
		}

		// Check if we've hit the safety limit
		if totalItemsProcessed >= MaxItemsToFetch {
			log.Printf("Reached maximum items limit (%d). Stopping pagination.", MaxItemsToFetch)
			break
		}

		// Update last item ID for the next page
		lastItemID = newLastItemID

		// Add a small delay to avoid overwhelming the server
		time.Sleep(RequestDelayMs * time.Millisecond)
	}

	log.Printf("Completed fetching all items. Processed %d items across %d pages.", totalItemsProcessed, totalPages)
	return nil
}

// fetchItemsPage fetches a single page of items based on the last item ID
func (s *CSGOSkinScraper) fetchItemsPage(lastItemID string) ([]models.CSGOSkinItem, string, error) {
	// Create HTTP request
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(CSGOSkinURL)
	req.Header.SetMethod("POST")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://csgoskin.ir")
	req.Header.Set("Referer", "https://csgoskin.ir/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Add required cookies
	// First, try to get from environment variables
	phpSessionID := os.Getenv("CSGOSKIN_PHPSESSID")
	userAuth := os.Getenv("CSGOSKIN_USERAUTH")

	// If not found in env vars, use the values provided as fallbacks
	if phpSessionID == "" {
		phpSessionID = "a655421c00d318211649a184c1f9fab7"
	}
	if userAuth == "" {
		userAuth = "RW81VTRGU1prcytZZElWUDhPRWltcDg2Wi9ieFhkQ0tNa09wRXZ1dTJBND06OhbdfXupnG4QWqWVL6tEN7c%3D"
	}

	// Set cookies
	req.Header.SetCookie("PHPSESSID", phpSessionID)
	req.Header.SetCookie("userauth", userAuth)

	// Set the payload with the lastItemID for pagination
	payload := fmt.Sprintf(`search={"knife":[],"tf2":[],"accessory":[],"pistol":[],"machineguns":[],"shotgun":[],"smg":[],"rifle":[],"sniperrifle":[],"fasttrade":1,"stattrack":0,"havesticker":0,"nametag":0,"FN":1,"MW":1,"FT":1,"WW":1,"BS":1,"minprice":0,"maxprice":0}&lastitem=%s`, lastItemID)

	req.SetBodyString(payload)

	// Send the request
	err := fasthttp.Do(req, resp)
	if err != nil {
		return nil, lastItemID, fmt.Errorf("request to CSGOSkin failed: %v", err)
	}

	// Debug the response if it's not 200
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, lastItemID, fmt.Errorf("CSGOSkin returned non-200 status code: %d, body: %s",
			resp.StatusCode(), string(resp.Body()))
	}

	// Parse the response
	var csgoItems []models.CSGOSkinItem
	if err := json.Unmarshal(resp.Body(), &csgoItems); err != nil {
		return nil, lastItemID, fmt.Errorf("failed to parse CSGOSkin response: %v", err)
	}

	// Get the last item ID for the next page
	newLastItemID := lastItemID
	if len(csgoItems) > 0 {
		newLastItemID = csgoItems[len(csgoItems)-1].ItemID
	}

	return csgoItems, newLastItemID, nil
}

// processItem processes a single item by inserting it into the database
func (s *CSGOSkinScraper) processItem(csgoItem models.CSGOSkinItem) error {
	// 1. First create or update the skin
	skin, err := s.convertToSkin(csgoItem)
	if err != nil {
		return fmt.Errorf("error converting skin: %v", err)
	}

	skinID, err := s.db.InsertSkin(skin)
	if err != nil {
		return fmt.Errorf("error inserting skin: %v", err)
	}

	// 2. Then create or update the specific item
	item, err := s.convertToItem(csgoItem, skinID)
	if err != nil {
		return fmt.Errorf("error converting item: %v", err)
	}

	_, err = s.db.InsertItem(item)
	if err != nil {
		return fmt.Errorf("error inserting item: %v", err)
	}

	return nil
}

// convertToSkin converts CSGOSkinItem to Skin model
func (s *CSGOSkinScraper) convertToSkin(csgoItem models.CSGOSkinItem) (*models.Skin, error) {
	// Extract category and subcategory
	category, subCategory := parseCategory(csgoItem.Name.Category)

	// Determine min and max float values based on quality
	// These are just approximations and should be refined with actual data
	minFloat, maxFloat := 0.0, 1.0
	quality := strings.TrimSpace(csgoItem.Quality)
	if quality == "Factory New" || quality == "Factory-New" {
		minFloat, maxFloat = 0.00, 0.07
	} else if quality == "Minimal Wear" || quality == "Minimal-Wear" {
		minFloat, maxFloat = 0.07, 0.15
	} else if quality == "Field-Tested" || quality == "Field-Tested" {
		minFloat, maxFloat = 0.15, 0.38
	} else if quality == "Well-Worn" || quality == "Well-Worn" {
		minFloat, maxFloat = 0.38, 0.45
	} else if quality == "Battle-Scarred" || quality == "Battle-Scarred" {
		minFloat, maxFloat = 0.45, 1.00
	}

	// Create the skin model
	skin := &models.Skin{
		MarketHashName: csgoItem.MarketHashName,
		Category:       category,
		SubCategory:    subCategory,
		SkinName:       strings.TrimSpace(csgoItem.Name.SkinName),
		IsStatTrak:     csgoItem.IsStatTrack == 1,
		Quality:        quality,
		MinFloat:       minFloat,
		MaxFloat:       maxFloat,
		IconURL:        "https://csgoskin.ir" + csgoItem.IconMedium,
	}

	return skin, nil
}

// convertToItem converts CSGOSkinItem to Item model
func (s *CSGOSkinScraper) convertToItem(csgoItem models.CSGOSkinItem, skinID string) (*models.Item, error) {
	// Parse float value
	var floatVal float64
	if csgoItem.Float != "" {
		var err error
		floatVal, err = strconv.ParseFloat(csgoItem.Float, 64)
		if err != nil {
			log.Printf("Warning: Could not parse float value %s: %v", csgoItem.Float, err)
		}
	}

	// Parse prices - NOTE: These are in Toman (IRT), not Rial (IRR)
	priceInToman, err := strconv.ParseFloat(csgoItem.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse price %s: %v", csgoItem.Price, err)
	}

	// Convert Toman to Rial (1 Toman = 10 Rial)
	priceInRial := priceInToman * 10

	var priceFailedInRial float64
	if csgoItem.PriceFailed != "" {
		priceFailedInToman, err := strconv.ParseFloat(csgoItem.PriceFailed, 64)
		if err != nil {
			log.Printf("Warning: Could not parse failed price %s: %v", csgoItem.PriceFailed, err)
		} else {
			// Convert Toman to Rial
			priceFailedInRial = priceFailedInToman * 10
		}
	}

	// Get the current IRR to USD conversion rate
	irrToUsdRate := GetIRRtoUSDRate()

	// Convert prices to USD (using the Rial value)
	priceUSD := priceInRial * irrToUsdRate

	// Parse Steam price
	var steamPriceUSD float64
	if csgoItem.PriceSteam != "" {
		// Remove $ sign and convert to float
		steamPriceStr := strings.TrimPrefix(csgoItem.PriceSteam, "$")
		steamPriceUSD, err = strconv.ParseFloat(steamPriceStr, 64)
		if err != nil {
			log.Printf("Warning: Could not parse steam price %s: %v", csgoItem.PriceSteam, err)
		}
	}

	// Create Item model
	item := &models.Item{
		SkinID:        skinID,
		MarketplaceID: s.marketplaceID,
		Float:         floatVal,
		Stickers:      csgoItem.Stickers,
		Price:         priceInRial,       // Store the price in Rial
		PriceFailed:   priceFailedInRial, // Store the failed price in Rial
		PriceUSD:      priceUSD,
		SteamPriceUSD: steamPriceUSD,
		Tradeable:     csgoItem.Tradeable,
		IsFastSell:    true, // Assuming all items from this query are fast sell since fasttrade=1
		MarketItemID:  csgoItem.ItemID,
	}

	return item, nil
}

// parseCategory extracts category and subcategory from a string
func parseCategory(fullCategory string) (string, string) {
	// Existing code...
	fullCategory = strings.TrimSpace(fullCategory)

	// Handle common categories
	if strings.Contains(fullCategory, "KNIFE") {
		return "Knife", fullCategory
	}

	if strings.Contains(fullCategory, "GLOVES") {
		return "Gloves", fullCategory
	}

	// For guns and other items
	parts := strings.Fields(fullCategory)
	if len(parts) > 0 {
		subCategory := parts[0]

		// Determine main category
		if contains([]string{"AK-47", "M4A4", "M4A1-S", "FAMAS", "GALIL", "AUG", "SG"}, subCategory) {
			return "Rifle", subCategory
		}

		if contains([]string{"AWP", "SCAR-20", "G3SG1", "SSG", "SSG 08"}, subCategory) {
			return "Sniper Rifle", subCategory
		}

		if contains([]string{"P90", "MP5", "MP7", "MP9", "MAC-10", "UMP-45", "PP-BIZON"}, subCategory) {
			return "SMG", subCategory
		}

		if contains([]string{"GLOCK", "USP-S", "P2000", "P250", "FIVE-SEVEN", "TEC-9", "CZ75", "DESERT EAGLE", "DUAL BERETTAS", "R8"}, subCategory) {
			return "Pistol", subCategory
		}

		if contains([]string{"NOVA", "XM1014", "MAG-7", "SAWED-OFF"}, subCategory) {
			return "Shotgun", subCategory
		}

		if contains([]string{"M249", "NEGEV"}, subCategory) {
			return "Machine Gun", subCategory
		}

		return "Other", subCategory
	}

	return "Unknown", fullCategory
}

// contains checks if a string is in a slice
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
