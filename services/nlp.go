package services

import (
	"log"

	"github.com/jdkato/prose/v2"
)

// ExtractCityNLP attempts to extract a city/location using NLP
func ExtractCityNLP(text string) string {
	doc, err := prose.NewDocument(text)
	if err != nil {
		log.Println("Error creating NLP document:", err)
		return ""
	}

	for _, ent := range doc.Entities() {
		if ent.Label == "GPE" { // Geo-Political Entity — countries, cities, etc.
			return ent.Text
		}
	}

	log.Println("No city found via NLP")
	return ""
}
