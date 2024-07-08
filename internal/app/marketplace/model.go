package marketplace

import (
	"bot/internal/app/core"
	"time"
)

type Product struct {
	core.Model
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ScrapedAt      time.Time
	Slug           string
	TelegramChatId int
	TelegramUserId int
	Url            string
	Marketplace    Marketplace
	Title          string
	ThresholdPrice int
	CurrentPrice   int
	OutOfStock     bool
}

func (p *Product) GetScrapedAt() time.Time {
	return p.ScrapedAt
}

func (p *Product) GetSlug() string {
	return p.Slug
}

func (p *Product) GetTelegramChatId() int {
	return p.TelegramChatId
}

func (p *Product) GetTelegramUserId() int {
	return p.TelegramUserId
}

func (p *Product) GetUrl() string {
	return p.Url
}

func (p *Product) GetMarketplace() Marketplace {
	return p.Marketplace
}

func (p *Product) GetTitle() string {
	return p.Title
}

func (p *Product) GetThresholdPrice() int {
	return p.ThresholdPrice
}

func (p *Product) GetCurrentPrice() int {
	return p.CurrentPrice
}

func (p *Product) IsOutOfStock() bool {
	return p.OutOfStock
}
