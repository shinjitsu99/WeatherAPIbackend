package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GoFiber/routes"
	"GoFiber/services"
	"GoFiber/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Initialize Redis first since translation service depends on it
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Verify Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Printf("Warning: Redis connection failed - %v", err)
	} else {
		log.Println("Successfully connected to Redis")
	}

	// Initialize services with Redis client
	services.InitRedis(rdb)

	// Set Google API key
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("API key not found in environment variables")
	}
	services.GoogleAPIKey = apiKey

	// Initialize database
	utils.InitDB()
	if utils.DB == nil {
		log.Fatal("Database connection is nil")
	}

	// Initialize HTTP client
	client := httpclient.NewClient(httpclient.WithHTTPTimeout(15 * time.Second))

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Prefork:       false,
		ServerHeader:  "Fiber",
		AppName:       "Weather API v1.0",
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  10 * time.Second,
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		status := fiber.Map{
			"status":  "ok",
			"version": "1.0",
		}

		if db, _ := utils.DB.DB(); db != nil {
			status["db"] = db.Ping() == nil
		} else {
			status["db"] = false
		}

		status["redis"] = rdb.Ping(c.Context()).Err() == nil
		return c.JSON(status)
	})

	// Set up routes
	routes.WeatherRoute(app, client)
	routes.QueryRoute(app, apiKey, client, rdb)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Graceful shutdown setup
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-shutdownChan
	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	log.Println("Server shut down successfully")
}
