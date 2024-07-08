package marketplace

import (
	"bot/internal/app/logger"
	"sync"
)

const WatcherIntervalInMinutes = 30

type WatcherResult struct {
	Original ProductDto
	Scraped  ProductDto
}

type Watcher struct {
	scraper Scraper
	service Service
	logger  logger.LoggerInterface
	locker  sync.Mutex
}

func NewWatcher(service Service, logger logger.LoggerInterface) Watcher {
	return Watcher{
		scraper: NewScraper(logger),
		service: service,
		logger:  logger,
	}
}

func (w *Watcher) Run(channel chan<- WatcherResult) error {
	w.locker.Lock()
	defer w.locker.Unlock()

	w.logger.Println("Running watcher...")

	scrapedCount := 0
	page := 1

	for {
		result := w.service.FindOutdatedPaginated(WatcherIntervalInMinutes, page, PerPageDefault)

		if result.Total == 0 {
			w.logger.Println("Watcher complete, nothing to scrape")
			return nil
		}

		page++

		for _, item := range result.Items {
			original := item.(Product)
			scraped, err := w.scraper.Scrape(original.Url)

			if err != nil && err != ErrOutOfStock {
				w.logger.Println("Unable to scrape:", err)
				continue
			}

			new := original

			new.ScrapedAt = scraped.GetScrapedAt()
			new.CurrentPrice = scraped.GetCurrentPrice()
			new.OutOfStock = scraped.IsOutOfStock()

			if (scraped.GetCurrentPrice() > 0) && (new.GetThresholdPrice() != scraped.GetCurrentPrice()) {
				new.ThresholdPrice = scraped.GetCurrentPrice()
			}

			w.service.Update(original.Id, &new)

			channel <- WatcherResult{
				Original: &original,
				Scraped:  scraped,
			}

			scrapedCount++
		}

		if result.IsLastPage() {
			break
		}
	}

	w.logger.Println("Watcher complete, scraped", scrapedCount, "products")

	return nil
}
