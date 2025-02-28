package testmodels

import "github.com/go-openapi/strfmt"

type RatingSystem struct {

	// Timestamp when the rating system was created.
	// Required: true
	// Format: date-time
	CreatedAt *strfmt.DateTime `json:"CreatedAt"`

	// A description of the rating system.
	// Required: true
	Description *string `json:"Description"`

	// Unique identifier for the rating system.
	// Required: true
	ID *string `json:"Id"`

	// Name of the rating system.
	// Required: true
	Name *string `json:"Name"`

	// site Url
	SiteURL string `json:"SiteUrl,omitempty"`

	// Timestamp when the rating system was last updated.
	// Required: true
	// Format: date-time
	UpdatedAt *strfmt.DateTime `json:"UpdatedAt"`
}
