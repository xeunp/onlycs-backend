package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	// New API URL for exchange rate
	ExchangeRateURL = "https://api.exnovinmarket.com/v2/tokens/status"
	// Convert Toman to Rial (1 Toman = 10 Rial)
	TomanToRialRate = 10
	// Default rate to use if API call fails
	DefaultUSDTtoIRRRate = 872000 * TomanToRialRate // 8,720,000 Rial
	// How often to refresh the rate (every 1 hour)
	ExchangeRateRefreshInterval = 1 * time.Hour
)

var (
	// Cache the exchange rate to avoid too many requests
	cachedUSDTtoIRRRate      float64
	cachedUSDTtoIRRRateMutex sync.RWMutex
	lastFetchTime            time.Time
)

// TokenStatus represents the structure of each token in the API response
type TokenStatus struct {
	Symbol            string  `json:"symbol"`
	ConvertRateInBase float64 `json:"convertRateInBase"`
	LastPriceInTMN    float64 `json:"lastPriceInTMN"`
	BestPrice         struct {
		Ask struct {
			Amount string  `json:"amount"`
			Price  float64 `json:"price"`
		} `json:"ask"`
		Bid struct {
			Amount string  `json:"amount"`
			Price  float64 `json:"price"`
		} `json:"bid"`
	} `json:"bestPrice"`
}

// GetUSDTtoIRRRate fetches the current USDT to IRR exchange rate
// Returns rate as Rial per 1 USDT
func GetUSDTtoIRRRate() float64 {
	cachedUSDTtoIRRRateMutex.RLock()
	// Check if we have a recent cached rate
	if cachedUSDTtoIRRRate > 0 && time.Since(lastFetchTime) < ExchangeRateRefreshInterval {
		rate := cachedUSDTtoIRRRate
		cachedUSDTtoIRRRateMutex.RUnlock()
		return rate
	}
	cachedUSDTtoIRRRateMutex.RUnlock()

	// Need to fetch a new rate
	cachedUSDTtoIRRRateMutex.Lock()
	defer cachedUSDTtoIRRRateMutex.Unlock()

	// Double-check in case another goroutine already updated while we were waiting for the lock
	if cachedUSDTtoIRRRate > 0 && time.Since(lastFetchTime) < ExchangeRateRefreshInterval {
		return cachedUSDTtoIRRRate
	}

	rate, err := fetchUSDTtoIRRRate()
	if err != nil {
		log.Printf("Error fetching USDT to IRR rate: %v, using default rate", err)
		if cachedUSDTtoIRRRate > 0 {
			// Keep using the old rate if available
			return cachedUSDTtoIRRRate
		}
		return DefaultUSDTtoIRRRate
	}

	// Update cache
	cachedUSDTtoIRRRate = rate
	lastFetchTime = time.Now()

	log.Printf("Updated USDT to IRR exchange rate: 1 USDT = %f IRR", rate)
	return rate
}

// fetchUSDTtoIRRRate fetches the current rate from the API
func fetchUSDTtoIRRRate() (float64, error) {
	// Create HTTP request
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(ExchangeRateURL)
	req.Header.SetMethod("GET")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	// Send the request
	err := fasthttp.Do(req, resp)
	if err != nil {
		return 0, fmt.Errorf("request to exchange rate API failed: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return 0, fmt.Errorf("exchange rate API returned non-200 status code: %d", resp.StatusCode())
	}

	// Parse the JSON response
	var tokens []TokenStatus
	if err := json.Unmarshal(resp.Body(), &tokens); err != nil {
		return 0, fmt.Errorf("failed to parse exchange rate API response: %v", err)
	}

	// Find the USDT token
	for _, token := range tokens {
		if token.Symbol == "USDT" {
			// Convert Toman to Rial (1 Toman = 10 Rial)
			priceInRial := token.LastPriceInTMN * TomanToRialRate
			return priceInRial, nil
		}
	}

	return 0, fmt.Errorf("couldn't find USDT token in the response")
}

// GetIRRtoUSDRate returns the conversion rate from IRR to USD
// (How many USD you get for 1 IRR)
func GetIRRtoUSDRate() float64 {
	usdtToIRR := GetUSDTtoIRRRate()
	if usdtToIRR <= 0 {
		return 1.0 / DefaultUSDTtoIRRRate
	}
	return 1.0 / usdtToIRR
}
