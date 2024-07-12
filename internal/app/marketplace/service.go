package marketplace

import (
	"bot/internal/app/core"
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"time"
)

type ProductDto interface {
	GetScrapedAt() time.Time
	GetSlug() string
	GetTelegramChatId() int
	GetTelegramUserId() int
	GetUrl() string
	GetMarketplace() Marketplace
	GetTitle() string
	GetThresholdPrice() int
	GetCurrentPrice() int
	IsOutOfStock() bool
}

type Repository interface {
	FindById(id int) (Product, error)
	FindOutdatedPaginated(offsetMinutes int, page int, perPage int) []Product
	GetCountOutdated(offsetMinutes int) int
	FindAllForUserPaginated(telegramChatId int, telegramUserId int, page int, perPage int) []Product
	GetCountForUser(telegramChatId int, telegramUserId int) int
	FindForUserByUrl(telegramChatId int, telegramUserId int, url string) (Product, error)
	FindForUserBySlug(telegramChatId int, telegramUserId int, slug string) (Product, error)
	Delete(id int) bool
	Save(model Product) (Product, error)
	IsUniqueSlug(slug string) bool
}

const PerPageDefault = 10

type Service struct {
	repository Repository
	logger     logger.LoggerInterface
}

func NewService(repository Repository, logger logger.LoggerInterface) Service {
	return Service{
		repository: repository,
		logger:     logger,
	}
}

func (s *Service) FindForUserByUrl(telegramChatId int, telegramUserId int, url string) (Product, error) {
	return s.repository.FindForUserByUrl(telegramChatId, telegramUserId, url)
}

func (s *Service) FindForUserBySlug(telegramChatId int, telegramUserId int, slug string) (Product, error) {
	return s.repository.FindForUserBySlug(telegramChatId, telegramUserId, slug)
}

func (s *Service) FindAllForUserPaginated(telegramChatId int, telegramUserId int, page int, perPage int) core.PaginatedResult {
	if perPage == 0 {
		perPage = PerPageDefault
	}

	models := s.repository.FindAllForUserPaginated(telegramChatId, telegramUserId, page, perPage)
	count := s.repository.GetCountForUser(telegramChatId, telegramUserId)

	items := make([]any, len(models))
	for i, model := range models {
		items[i] = model
	}

	return core.NewPaginatedResult(items, page, perPage, count)
}

func (s *Service) FindOutdatedPaginated(outdatedOffsetMinutes int, page int, perPage int) core.PaginatedResult {
	if perPage == 0 {
		perPage = PerPageDefault
	}

	models := s.repository.FindOutdatedPaginated(outdatedOffsetMinutes, page, perPage)
	count := s.repository.GetCountOutdated(outdatedOffsetMinutes)

	items := make([]any, len(models))
	for i, model := range models {
		items[i] = model
	}

	return core.NewPaginatedResult(items, page, perPage, count)
}

func (s *Service) Create(dto ProductDto) (Product, error) {
	model := Product{}

	return s.updateByDto(model, dto)
}

func (s *Service) Update(id int, dto ProductDto) (Product, error) {
	model, err := s.repository.FindById(id)
	if err != nil {
		return Product{}, err
	}

	return s.updateByDto(model, dto)
}

func (s *Service) Delete(id int) bool {
	model, err := s.repository.FindById(id)
	if err != nil {
		return false
	}

	return s.repository.Delete(model.Id)
}

func (s *Service) updateByDto(model Product, dto ProductDto) (Product, error) {
	model.ScrapedAt = dto.GetScrapedAt()
	model.TelegramChatId = dto.GetTelegramChatId()
	model.TelegramUserId = dto.GetTelegramUserId()
	model.Marketplace = dto.GetMarketplace()
	model.Url = dto.GetUrl()
	model.Title = dto.GetTitle()
	model.ThresholdPrice = dto.GetThresholdPrice()
	model.CurrentPrice = dto.GetCurrentPrice() // TODO: price history? (https://github.com/timescale/timescaledb)
	model.OutOfStock = dto.IsOutOfStock()

	if model.Slug == "" {
		model.Slug = s.getUniqueSlug()
	}

	model, err := s.repository.Save(model)
	if err != nil {
		s.logger.Println("Unable to save model:", err)
		return Product{}, err
	}

	return model, nil
}

func (s *Service) getUniqueSlug() string {
	var slug string

	for {
		slug = helpers.GetRandomString()

		if s.isUniqueSlug(slug) {
			break
		}
	}

	return slug
}

func (s *Service) isUniqueSlug(slug string) bool {
	return s.repository.IsUniqueSlug(slug)
}
