package marketplace

import (
	"regexp"
	"strings"
)

type Marketplace int

const (
	MarketplaceUnknown Marketplace = iota
	MarketplaceWildberries
	MarketplaceOzon
)

const (
	patternWildberries string = `^(https?://)?(www.)?(wildberries\.ru/catalog/\d+/detail\.aspx\??)(.*&)?(?:targetUrl=[A-Z]+)?(size=\d+)?(&.*)?$`
	patternOzon        string = `^(https?://)?(www.)?(ozon\.ru(/product/[a-z0-9-]+/|/t/[A-Za-z0-9-]+))(\?.+)?$`
)

// Detect marketplace type by URL.
func DetectMarketplaceByUrl(url string) Marketplace {
	if matched, _ := regexp.MatchString(patternWildberries, url); matched {
		return MarketplaceWildberries
	} else if matched, _ := regexp.MatchString(patternOzon, url); matched {
		return MarketplaceOzon
	}

	return MarketplaceUnknown
}

// Get clean marketplace URL.
func GetCleanUrl(url string) string {
	var pattern string

	replacement := "https://www.$3"
	marketplace := DetectMarketplaceByUrl(url)

	switch marketplace {
	case MarketplaceWildberries:
		pattern = patternWildberries
		replacement = "https://www.$3$5"
	case MarketplaceOzon:
		pattern = patternOzon
	}

	regex := regexp.MustCompile(pattern)

	return strings.TrimSuffix(regex.ReplaceAllString(url, replacement), "?")
}

func GetMarketplaceName(product ProductDto) string {
	switch product.GetMarketplace() {
	case MarketplaceOzon:
		return "Ozon"
	case MarketplaceWildberries:
		return "Wildberries"
	}

	return ""
}
