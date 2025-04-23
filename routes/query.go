package routes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"GoFiber/models"
	"GoFiber/services"
	"GoFiber/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	cacheTTL = 24 * time.Hour
)

func QueryRoute(app *fiber.App, apiKey string, client *httpclient.Client, rdb *redis.Client) {
	app.Post("/query", handleQuery(apiKey, client, rdb))
}

func handleQuery(apiKey string, client *httpclient.Client, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		startTime := time.Now()
		ctx := c.Context()

		var body struct {
			OriginalQuery string `json:"original_query"`
		}
		if err := c.BodyParser(&body); err != nil {
			log.Printf("Invalid request body: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid request format",
				"details": err.Error(),
			})
		}

		originalQuery := strings.TrimSpace(body.OriginalQuery)
		if originalQuery == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing or empty 'original_query' field",
			})
		}

		// Try cache first
		cacheResponse, cacheErr := tryCache(ctx, originalQuery, rdb)
		if cacheErr == nil {
			log.Printf("Request served from cache in %v", time.Since(startTime))
			return c.JSON(cacheResponse)
		} else if !errors.Is(cacheErr, redis.Nil) {
			log.Printf("Cache error: %v", cacheErr)
		}

		// In your main handler function:
		dbResponse, err := tryDatabase(ctx, originalQuery, rdb)
		if err != nil {
			log.Printf("Database error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error occurred",
			})
		}

		// Check if we got a valid response (not empty)
		if dbResponse["original_query"] != nil && dbResponse["translated_response"] != nil {
			log.Printf("Request served from database in %v", time.Since(startTime))
			return c.JSON(dbResponse)
		}

		// Process new request if not found in cache or database
		finalResponse, processErr := processNewRequest(ctx, originalQuery, apiKey, client, rdb)
		if processErr != nil {
			log.Printf("Processing error: %v", processErr)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": processErr.Error(),
			})
		}

		log.Printf("Request processed in %v", time.Since(startTime))
		return c.JSON(fiber.Map{
			"original_query":      originalQuery,
			"translated_response": finalResponse,
			"source":              "new_processing",
			"processing_time":     time.Since(startTime).String(),
		})
	}
}

func tryCache(ctx context.Context, query string, rdb *redis.Client) (fiber.Map, error) {
	cacheKey := fmt.Sprintf("query:%s", query)
	cached, err := rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	return fiber.Map{
		"original_query":      query,
		"translated_response": cached,
		"source":              "redis_cache",
	}, nil
}

func tryDatabase(ctx context.Context, query string, rdb *redis.Client) (fiber.Map, error) {
	var existing models.Translation
	err := utils.DB.WithContext(ctx).
		Where("original_text = ?", query).
		First(&existing).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty response when record not found
			return fiber.Map{}, nil
		}
		// Return other database errors
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Update cache in background if we have a valid record
	if rdb != nil && existing.ID != 0 {
		go func() {
			bgCtx := context.Background()
			cacheKey := fmt.Sprintf("query:%s", query)
			if err := rdb.Set(bgCtx, cacheKey, existing.TranslatedText, cacheTTL).Err(); err != nil {
				log.Printf("Failed to update cache: %v", err)
			}
		}()
	}

	return fiber.Map{
		"original_query":      existing.OriginalText,
		"translated_response": existing.TranslatedText,
		"source":              "database",
	}, nil
}

func processNewRequest(ctx context.Context, originalQuery, apiKey string, client *httpclient.Client, rdb *redis.Client) (string, error) {
	// Translation with nil check
	if client == nil {
		return "", errors.New("http client is nil")
	}

	translatedQuery, detectedLang, err := services.TranslateText(originalQuery, "en", client)
	if err != nil {
		return "", fmt.Errorf("translation failed: %v", err)
	}

	// Location extraction
	city := services.ExtractCityNLP(translatedQuery)
	if city == "" {
		return "", errors.New("could not extract city from query")
	}

	// Coordinates
	lat, lon, err := services.GetCoordinates(city, apiKey, client)
	if err != nil {
		return "", fmt.Errorf("coordinates lookup failed: %v", err)
	}

	// Weather
	weather, err := services.GetWeather(lat, lon, apiKey, client)
	if err != nil {
		return "", fmt.Errorf("weather lookup failed: %v", err)
	}

	// Prepare response
	englishResponse := fmt.Sprintf("The weather in %s is %.1f°C with %s.",
		city, weather.Temperature.Degrees, weather.Condition.Description.Text)

	var finalResponse string
	if detectedLang == "en" {
		finalResponse = englishResponse
	} else {
		finalResponse, _, err = services.TranslateText(englishResponse, detectedLang, client)
		if err != nil {
			log.Printf("Back-translation failed, using English: %v", err)
			finalResponse = englishResponse
		}
	}

	// Save to database in background
	go func() {
		bgCtx := context.Background()
		translation := models.Translation{
			OriginalText:   originalQuery,
			TranslatedText: finalResponse,
			SourceLang:     detectedLang,
			TargetLang:     "en",
			Timestamp:      time.Now(),
		}
		if err := utils.DB.WithContext(bgCtx).Create(&translation).Error; err != nil {
			log.Printf("Background database save failed: %v", err)
		}
	}()

	// Update cache
	if err := rdb.Set(ctx, fmt.Sprintf("query:%s", originalQuery), finalResponse, cacheTTL).Err(); err != nil {
		log.Printf("Cache update failed: %v", err)
	}

	return finalResponse, nil
}
