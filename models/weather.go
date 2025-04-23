package models

type WeatherResponse struct {
	Temperature struct {
		Degrees float64 `json:"degrees"`
		Unit    string  `json:"unit"`
	} `json:"temperature"`
	Condition struct {
		Description struct {
			Text string `json:"text"`
		} `json:"description"`
	} `json:"weatherCondition"`
}
