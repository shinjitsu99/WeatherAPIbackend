package models

import "time"

type Translation struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OriginalText   string    `json:"original_text"`
	TranslatedText string    `json:"translated_text"`
	SourceLang     string    `json:"source_lang"`
	TargetLang     string    `json:"target_lang"`
	Timestamp      time.Time `json:"timestamp"`
	Disliked       *bool     `json:"disliked"` // Optional, can be null
	Feedback       *string   `json:"feedback"` // Optional, can be null
}
