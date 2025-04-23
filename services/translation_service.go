package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient  *redis.Client
	GoogleAPIKey string
)

func InitRedis(rdb *redis.Client) {
	redisClient = rdb
}

func TranslateText(text, targetLang string, client *httpclient.Client) (string, string, error) {
	// Create cache key first so we can use it throughout the function
	cacheKey := fmt.Sprintf("translation:%s:%s", text, targetLang)

	// First try Redis cache if available
	if redisClient != nil {
		ctx := context.Background()

		if cached, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
			var result struct {
				TranslatedText string `json:"translatedText"`
				DetectedLang   string `json:"detectedLang"`
			}
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return result.TranslatedText, result.DetectedLang, nil
			}
		}
	}

	// Fall back to direct API call
	translatedText, detectedLang, err := fetchTranslationWithDetection(text, targetLang)
	if err != nil {
		return "", "", fmt.Errorf("translation failed: %w", err)
	}

	// Cache the result if Redis is available
	if redisClient != nil {
		ctx := context.Background()
		result := struct {
			TranslatedText string `json:"translatedText"`
			DetectedLang   string `json:"detectedLang"`
		}{
			TranslatedText: translatedText,
			DetectedLang:   detectedLang,
		}

		if jsonBytes, err := json.Marshal(result); err == nil {
			if err := redisClient.Set(ctx, cacheKey, jsonBytes, 24*time.Hour).Err(); err != nil {
				log.Printf("Failed to cache translation: %v", err)
			}
		}
	}

	return translatedText, detectedLang, nil
}

func fetchTranslationWithDetection(text, targetLang string) (string, string, error) {
	url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2?key=%s", GoogleAPIKey)

	requestBody := map[string]interface{}{
		"q":      text,
		"target": targetLang,
		"format": "text",
	}
	reqBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("translation API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText         string `json:"translatedText"`
				DetectedSourceLanguage string `json:"detectedSourceLanguage"`
			} `json:"translations"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse translation response: %w", err)
	}

	if len(result.Data.Translations) == 0 {
		return "", "", fmt.Errorf("no translations returned")
	}

	translation := result.Data.Translations[0]
	translatedText := htmlUnescape(translation.TranslatedText)

	// Clean up trailing punctuation
	if !strings.HasSuffix(strings.TrimSpace(text), "?") && strings.HasSuffix(translatedText, "?") {
		translatedText = strings.TrimSuffix(translatedText, "?")
	}

	return translatedText, translation.DetectedSourceLanguage, nil
}

func htmlUnescape(s string) string {
	replacer := strings.NewReplacer(
		"&quot;", "\"",
		"&#39;", "'",
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
	)
	return replacer.Replace(s)
}
