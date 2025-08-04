package main

import (
	"github.com/joho/godotenv"
	"github.com/mswatii/cs2-arbitrage/internal/api"
	"github.com/mswatii/cs2-arbitrage/internal/database"
	"github.com/mswatii/cs2-arbitrage/internal/scraper"
	"github.com/valyala/fasthttp"
	"log"
	"os"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or cannot be loaded")
	}

	// Connect to database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create tables if they don't exist
	if err := db.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Initialize API handler
	handler := api.NewHandler(db)

	// Initialize exchange rate (this will cache the first value)
	exchangeRate := scraper.GetUSDTtoIRRRate()
	log.Printf("Initial USDT to IRR exchange rate: %f", exchangeRate)

	// Run the scraper on startup if SKIP_INITIAL_SCRAPE is not set
	if os.Getenv("SKIP_INITIAL_SCRAPE") != "true" {
		log.Println("Starting initial data scrape...")
		csgoSkinScraper, err := scraper.NewCSGOSkinScraper(db)
		if err != nil {
			log.Printf("Warning: Failed to initialize scraper: %v", err)
		} else {
			// Run the scraper in a goroutine so it doesn't block server startup
			go func() {
				if err := csgoSkinScraper.FetchItems(); err != nil {
					log.Printf("Error during initial data scrape: %v", err)
				} else {
					log.Println("Initial data scrape completed successfully")
				}
			}()
		}
	} else {
		log.Println("Skipping initial data scrape (SKIP_INITIAL_SCRAPE=true)")
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := fasthttp.ListenAndServe(":"+port, handler.HandleRequest); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
