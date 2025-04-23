package routes

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gojek/heimdall/v7/httpclient"

	"GoFiber/services"
)

func WeatherRoute(app *fiber.App, client *httpclient.Client) {
	app.Post("/weather", func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid input")
		}

		city := body["city"]
		apiKey := os.Getenv("GOOGLE_API_KEY")

		lat, lon, err := services.GetCoordinates(city, apiKey, client)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to get coordinates")
		}

		weather, err := services.GetWeather(lat, lon, apiKey, client)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch weather data")
		}

		return c.JSON(fiber.Map{
			"city":        city,
			"temperature": weather.Temperature.Degrees,
			"condition":   weather.Condition.Description.Text,
		})
	})
}
