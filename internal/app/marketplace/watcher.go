package marketplace

import (
	"bot/internal/app/logger"
	"sync"
)

type WatcherResult struct {
	Original ProductDto
	Scraped  ProductDto
}

type Watcher struct {
	scraper           Scraper
	service           Service
	logger            logger.LoggerInterface
	locker            sync.Mutex
	intervalInMinutes int
}

func NewWatcher(service Service, logger logger.LoggerInterface, intervalInMinutes int, timeout int) Watcher {
	return Watcher{
		scraper:           NewScraper(logger, timeout),
		service:           service,
		logger:            logger,
		intervalInMinutes: intervalInMinutes,
	}
}

func (w *Watcher) Run(channel chan<- WatcherResult) error {
	w.locker.Lock()
	defer w.locker.Unlock()

	w.logger.Println("Running watcher...")

	scrapedCount := 0
	page := 1

	result := w.service.FindOutdatedPaginated(w.intervalInMinutes, page, PerPageDefault)

	if result.Total == 0 {
		w.logger.Println("Watcher complete, nothing to scrape")
		return nil
	}

	w.logger.Println("Watching", result.Total, "item(s)")

	pageIterationCount := 0

	for {
		if pageIterationCount == len(result.Items) {
			pageIterationCount = 0
			page++
			result = w.service.FindOutdatedPaginated(w.intervalInMinutes, page, PerPageDefault)
		}

		if len(result.Items) == 0 {
			break
		}

		w.logger.Println("Page", page, "/", result.LastPage)

		for i, item := range result.Items {
			original := item.(Product)

			w.logger.Println("Item", (i + 1), "-", original.GetUrl())

			scraped, err := w.scraper.Scrape(original.Url)

			if err != nil && err != ErrOutOfStock {
				pageIterationCount++
				scrapedCount++

				w.logger.Println("Unable to scrape:", err)

				continue
			}

			new := original

			new.ScrapedAt = scraped.GetScrapedAt()
			new.OutOfStock = scraped.IsOutOfStock()

			if scraped.GetCurrentPrice() > 0 {
				new.CurrentPrice = scraped.GetCurrentPrice()

				if new.GetThresholdPrice() != scraped.GetCurrentPrice() {
					new.ThresholdPrice = scraped.GetCurrentPrice()
				}
			}

			w.service.Update(original.Id, &new)

			channel <- WatcherResult{
				Original: &original,
				Scraped:  scraped,
			}

			pageIterationCount++
			scrapedCount++
		}

		if result.IsLastPage() {
			break
		}
	}

	w.logger.Println("Watcher complete, scraped", scrapedCount, "products")

	return nil
}
