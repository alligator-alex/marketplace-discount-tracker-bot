package marketplace

import (
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"context"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	chromedpUndetected "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type ScrapedProduct struct {
	context.Context
	url         string
	marketplace Marketplace
	title       string
	price       int
	outOfStock  bool
}

func (p *ScrapedProduct) GetScrapedAt() time.Time {
	return time.Now()
}

func (p *ScrapedProduct) GetSlug() string {
	return ""
}

func (p *ScrapedProduct) GetTelegramChatId() int {
	return 0
}

func (p *ScrapedProduct) GetTelegramUserId() int {
	return 0
}

func (p *ScrapedProduct) GetUrl() string {
	return p.url
}

func (p *ScrapedProduct) GetMarketplace() Marketplace {
	return p.marketplace
}

func (p *ScrapedProduct) GetTitle() string {
	return p.title
}

func (p *ScrapedProduct) GetCurrentPrice() int {
	return p.price
}

func (p *ScrapedProduct) GetThresholdPrice() int {
	return 0
}

func (p *ScrapedProduct) IsOutOfStock() bool {
	return p.outOfStock
}

var ErrEmptyUrl = errors.New("empty marketplace url")
var ErrUnsupported = errors.New("unsupported marketplace")
var ErrOutOfStock error = errors.New("product out of stock")
var ErrNotFound error = errors.New("product not found")

type Scraper struct {
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewScraper(logger logger.LoggerInterface) Scraper {
	return Scraper{
		logger: logger,
	}
}

// Create new browser instance.
func (s *Scraper) newBrowserInstance() (context.Context, context.CancelFunc, error) {
	var instance context.Context
	var cancel context.CancelFunc
	var err error

	instance, cancel, err = chromedpUndetected.New(chromedpUndetected.NewConfig(
		chromedpUndetected.WithHeadless(),
		//chromedpUndetected.WithTimeout(time.Second*15),
	))

	if err != nil {
		return nil, nil, err
	}

	return instance, cancel, nil
}

// Scrape target URL.
func (s *Scraper) Scrape(url string) (ProductDto, error) {
	if url == "" {
		return &ScrapedProduct{}, ErrEmptyUrl
	}

	var cancel context.CancelFunc
	var err error

	s.ctx, cancel, err = s.newBrowserInstance()
	if err != nil {
		s.logger.Println("Unable to initialize browser", err)
		return &ScrapedProduct{}, err
	}

	defer cancel()

	switch DetectMarketplaceByUrl(url) {
	case MarketplaceWildberries:
		return s.scrapeWildberries(url)
	case MarketplaceOzon:
		return s.scrapeOzon(url)
	}

	return &ScrapedProduct{}, ErrUnsupported
}

// Scrape many URLs one by one.
func (s *Scraper) ScrapeMany(urls []string) ([]ProductDto, error) {
	if len(urls) < 1 {
		return nil, ErrEmptyUrl
	}

	var cancel context.CancelFunc
	var err error

	s.ctx, cancel, err = s.newBrowserInstance()
	if err != nil {
		s.logger.Println("Unable to initialize browser", err)
		return nil, err
	}

	defer cancel()

	var item ProductDto
	var items []ProductDto
	var scrapedProduct *ScrapedProduct

	for i, url := range urls {
		// pause between urls to avoid blocking
		if i > 0 {
			s.logger.Println("Cooldown 2 seconds...")
			time.Sleep(2 * time.Second)
		}

		switch DetectMarketplaceByUrl(url) {
		case MarketplaceWildberries:
			item, err = s.scrapeWildberries(url)
		case MarketplaceOzon:
			item, err = s.scrapeOzon(url)
		default:
			item, err = &ScrapedProduct{}, ErrUnsupported
		}

		scrapedProduct = item.(*ScrapedProduct)

		if s.isUnknownError(err) {
			s.logger.Println("Unknown error while scraping URL:", err)
			continue
		}

		if err == ErrUnsupported {
			s.logger.Println("Unsupported URL:", url)
			continue
		}

		if err == ErrNotFound {
			s.logger.Println("Nothing found at URL:", url)
			continue
		}

		s.logger.Println("Found:", scrapedProduct.GetTitle(), " Price:", scrapedProduct.GetCurrentPrice(), "Out of stock?", scrapedProduct.IsOutOfStock())

		items = append(items, scrapedProduct)
	}

	return items, nil
}

// Scrape Wildberries.
func (s *Scraper) scrapeWildberries(url string) (ProductDto, error) {
	product := &ScrapedProduct{
		url:         url,
		marketplace: MarketplaceWildberries,
	}

	s.logger.Println("Scraping Wildberries URL:", url)

	pageContext, cancel := chromedp.NewContext(s.ctx)
	defer cancel()

	pageContext, cancel = context.WithTimeout(pageContext, 15*time.Second)
	defer cancel()

	err := s.runWithActions(
		pageContext,
		chromedp.Navigate(url),
		chromedp.WaitNotVisible(".general-preloader"),

		// check if error page
		chromedp.QueryAfter(".content404", func(ctx context.Context, execCtx runtime.ExecutionContextID, nodes ...*cdp.Node) error {
			if len(nodes) < 1 {
				return nil
			}

			return ErrNotFound
		}, chromedp.ByQuery, chromedp.AtLeast(0)),

		// check if out of stock
		chromedp.QueryAfter(".product-page", func(ctx context.Context, id runtime.ExecutionContextID, nodes ...*cdp.Node) error {
			if len(nodes) < 1 {
				return nil
			}

			var isOutOfStock bool
			outOfStockJS := `function () {
				return (this.querySelector('.sold-out-product') !== null);
			}`

			s.callFunctionOnNode(ctx, nodes[0], outOfStockJS, &isOutOfStock)

			if !isOutOfStock {
				return nil
			}

			var title string
			titleJS := `function () {
				return this.querySelector('h1').innerText;
			}`

			s.callFunctionOnNode(ctx, nodes[0], titleJS, &title)

			title = strings.TrimSpace(title)
			if title == "" {
				return ErrNotFound
			}

			product.title = title
			product.outOfStock = true

			return ErrOutOfStock
		}, chromedp.ByQuery, chromedp.AtLeast(0)),

		// get product title
		chromedp.Text("h1", &product.title, chromedp.ByQuery),

		// get product price
		chromedp.QueryAfter(".price-block__final-price", func(ctx context.Context, id runtime.ExecutionContextID, nodes ...*cdp.Node) error {
			if len(nodes) < 1 {
				return nil
			}

			var price string
			priceJS := `function () { 
				return this.innerHTML;
			}`

			s.callFunctionOnNode(ctx, nodes[0], priceJS, &price)

			product.price = helpers.CurrencyToMinor(s.parsePrice(price))

			return nil
		}, chromedp.ByQuery, chromedp.AtLeast(0)),

		// add selected size to the product title (if available)
		chromedp.QueryAfter(".sizes-list__button.active", func(ctx context.Context, id runtime.ExecutionContextID, nodes ...*cdp.Node) error {
			if len(nodes) < 1 {
				return nil
			}

			var size string
			sizeJS := `function () {
				return this.querySelector(".sizes-list__size").innerHTML;
			}`

			s.callFunctionOnNode(ctx, nodes[0], sizeJS, &size)

			size = strings.TrimSpace(size)
			if size != "" {
				product.title = helpers.ConcatStrings(product.title, " ", size)
			}

			return nil
		}, chromedp.ByQuery, chromedp.AtLeast(0)),
	)

	if s.isUnknownError(err) {
		s.logger.Println("Unknown error while scraping Wildberries URL:", err)
		return &ScrapedProduct{}, err
	}

	return product, err
}

// Scrape Ozon.
func (s *Scraper) scrapeOzon(url string) (ProductDto, error) {
	product := &ScrapedProduct{
		url:         url,
		marketplace: MarketplaceOzon,
	}

	s.logger.Println("Scraping Ozon URL:", url)

	pageContext, cancel := chromedp.NewContext(s.ctx)
	defer cancel()

	pageContext, cancel = context.WithTimeout(pageContext, 15*time.Second)
	defer cancel()

	err := s.runWithActions(
		pageContext,
		chromedp.Navigate(url),
		chromedp.WaitVisible("[data-widget=\"container\"]"),

		chromedp.QueryAfter("[data-widget=\"container\"]", func(ctx context.Context, id runtime.ExecutionContextID, nodes ...*cdp.Node) error {
			// check if error page
			var hasError bool
			errorPageJS := `function () {
				return (this.querySelector('[data-widget="error"]') !== null);
			}`

			s.callFunctionOnNode(ctx, nodes[0], errorPageJS, &hasError)

			if hasError {
				return ErrNotFound
			}

			// check if out of stock
			var isOutOfStock bool
			outOfStockJS := `function () {
				return (this.querySelector('[data-widget="webOutOfStock"]') !== null);
			}`

			s.callFunctionOnNode(ctx, nodes[0], outOfStockJS, &isOutOfStock)

			if isOutOfStock {
				var title string
				titleJS := `function () {
					return this.querySelector('p').innerText;
				}`

				s.callFunctionOnNode(ctx, nodes[0], titleJS, &title)

				title = strings.TrimSpace(title)
				if title == "" {
					return ErrNotFound
				}

				product.title = title
				product.outOfStock = true

				return ErrOutOfStock
			}

			// scrape product page
			var title string
			titleJS := `function () {
				return this.querySelector('h1').innerText;
			}`

			s.callFunctionOnNode(ctx, nodes[0], titleJS, &title)

			product.title = strings.TrimSpace(title)

			// there could be "with ozon card" button with "fake" price
			var price string
			priceJS := `function () {
				let nodes = this.querySelector('[data-widget="webPrice"]').querySelectorAll('span:first-of-type');

				return (nodes.length > 1)
					? nodes[3].innerText
					: nodes[0].innerText;
			}`

			s.callFunctionOnNode(ctx, nodes[0], priceJS, &price)

			product.price = helpers.CurrencyToMinor(s.parsePrice(price))

			return nil
		}, chromedp.ByQuery, chromedp.NodeVisible),
	)

	if s.isUnknownError(err) {
		s.logger.Println("Unknown error while scraping Ozon URL:", err)
		return &ScrapedProduct{}, err
	}

	return product, err
}

// Run browser with actions to scrape product.
func (s *Scraper) runWithActions(ctx context.Context, actions ...chromedp.Action) error {
	return chromedp.Run(ctx, actions...)
}

// Check if error is unknown to the system.
func (s *Scraper) isUnknownError(err error) bool {
	if err == nil {
		return false
	}

	knownErrors := []error{
		ErrEmptyUrl,
		ErrUnsupported,
		ErrOutOfStock,
		ErrNotFound,
	}

	for _, knownError := range knownErrors {
		if err == knownError {
			return false
		}
	}

	return true
}

// Parse price from HTML string.
func (s *Scraper) parsePrice(price string) float64 {
	reg, err := regexp.Compile("[^0-9,.]+")
	if err != nil {
		s.logger.Println("Unable to compile regex for price:", err)
		return 0
	}

	price = reg.ReplaceAllString(price, "")
	price = strings.ReplaceAll(price, ",", ".")

	priceValue, err := strconv.ParseFloat(strings.TrimSpace(price), 64)
	if err != nil {
		s.logger.Println("Unable to parse price:", price)
		return 0
	}

	return math.Round(priceValue*100) / 100
}

// Copy of chromedp.callFunctionOnNode().
// https://github.com/chromedp/chromedp/blob/master/query.go#L439
func (s *Scraper) callFunctionOnNode(ctx context.Context, node *cdp.Node, function string, res interface{}, args ...interface{}) error {
	r, err := dom.ResolveNode().WithNodeID(node.NodeID).Do(ctx)
	if err != nil {
		return err
	}

	err = chromedp.CallFunctionOn(function, res, func(p *runtime.CallFunctionOnParams) *runtime.CallFunctionOnParams {
		return p.WithObjectID(r.ObjectID)
	}, args...).Do(ctx)

	if err != nil {
		return err
	}

	_ = runtime.ReleaseObject(r.ObjectID).Do(ctx)

	return nil
}
