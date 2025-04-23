package utils

import (
	"time"

	"github.com/gojek/heimdall/v7/httpclient"
)

// NewHTTPClient creates a new HTTP client with Heimdall
func NewHTTPClient() *httpclient.Client {
	return httpclient.NewClient(
		httpclient.WithHTTPTimeout(5 * time.Second), // Timeout for each request
	)
}
