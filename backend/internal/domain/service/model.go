package service

import (
	"time"
)

// Category of services.
type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// Service represents a priced service line.
type Service struct {
	ID              int64     `json:"id"`
	CategoryID      int64     `json:"category_id"`
	CategorySlug    string    `json:"category_slug,omitempty"`
	CategoryName    string    `json:"category_name,omitempty"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	LongDescription string    `json:"long_description"`
	PriceCents      int64     `json:"price_cents"`
	PriceRUB        int64     `json:"price"` // derived price in major units
	Currency        string    `json:"currency"`
	DurationMin     int       `json:"duration_minutes"`
	IsFromPrice     bool      `json:"is_from_price"`
	IsActive        bool      `json:"is_active"`
	Archived        bool      `json:"archived"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
