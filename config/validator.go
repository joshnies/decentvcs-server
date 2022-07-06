package config

import "github.com/go-playground/validator/v10"

var Validator *validator.Validate

// Initialize validator instance.
// A single instance is used to utilize the validator package's caching feature.
func InitValidator() {
	Validator = validator.New()
}
