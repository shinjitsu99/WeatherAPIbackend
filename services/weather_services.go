package services

import (
	"GoFiber/models"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gojek/heimdall/v7/httpclient"
)

func GetCoordinates(city, apiKey string, client *httpclient.Client) (float64, float64, error) {
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", city, apiKey)
	resp, err := client.Get(url, nil)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geo models.GeocodeResponse
	if err := json.Unmarshal(body, &geo); err != nil {
		return 0, 0, err
	}

	if len(geo.Results) == 0 {
		return 0, 0, fmt.Errorf("no results found for city: %s", city)
	}

	loc := geo.Results[0].Geometry.Location
	return loc.Lat, loc.Lng, nil
}

func GetWeather(lat, lon float64, apiKey string, client *httpclient.Client) (*models.WeatherResponse, error) {
	url := fmt.Sprintf("https://weather.googleapis.com/v1/currentConditions:lookup?key=%s&location.latitude=%f&location.longitude=%f", apiKey, lat, lon)
	resp, err := client.Get(url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var weather models.WeatherResponse
	if err := json.Unmarshal(body, &weather); err != nil {
		return nil, err
	}

	return &weather, nil
}
