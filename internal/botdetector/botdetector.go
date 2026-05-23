package botdetector

import (
	"regexp"
	"strings"
)

// Single combined regex for all known bot patterns — one MatchString call instead of 40+
var botPattern = regexp.MustCompile(`(?i)` +
	// Search engine bots
	`googlebot|bingbot|yahoo|duckduckbot|baiduspider|yandex|slurp|` +
	// Social media bots
	`facebookexternalhit|twitterbot|linkedinbot|whatsapp|telegrambot|discordbot|slackbot|` +
	// SEO/Monitoring bots
	`ahrefsbot|semrushbot|mj12bot|dotbot|rogerbot|screaming frog|sitebulb|` +
	// Generic bot indicators
	`\bbot\b|\bcrawler\b|\bspider\b|\bscraper\b|\bfetcher\b|` +
	// Headless browsers
	`headlesschrome|phantomjs|selenium|webdriver|puppeteer|` +
	// Monitoring services
	`pingdom|uptimerobot|newrelic|statuscake|sitechecker|` +
	// Archiving/Indexing
	`archive\.org|ia_archiver|wayback`)

// Separate regex for patterns that must match at start of string
var httpLibPattern = regexp.MustCompile(`(?i)^(curl|wget|python-requests|go-http-client|axios|httpie)`)

// Additional suspicious patterns
var suspiciousPatterns = []string{
	"http",
	"library",
	"fetcher",
	"monitoring",
	"check",
}

// IsBot determines if a user agent string belongs to a bot
func IsBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}

	ua := strings.TrimSpace(userAgent)

	// Single regex check for all known bot patterns
	if botPattern.MatchString(ua) {
		return true
	}

	// Check HTTP library patterns (anchored to start)
	if httpLibPattern.MatchString(ua) {
		return true
	}

	uaLower := strings.ToLower(ua)

	// Very short user agents are often bots
	if len(ua) < 20 {
		for _, suspicious := range suspiciousPatterns {
			if strings.Contains(uaLower, suspicious) {
				return true
			}
		}
	}

	// Check for missing common browser indicators
	hasCommonBrowser := strings.Contains(uaLower, "mozilla") ||
		strings.Contains(uaLower, "chrome") ||
		strings.Contains(uaLower, "safari") ||
		strings.Contains(uaLower, "firefox") ||
		strings.Contains(uaLower, "edge")

	if !hasCommonBrowser {
		for _, suspicious := range suspiciousPatterns {
			if strings.Contains(uaLower, suspicious) {
				return true
			}
		}
	}

	return false
}

// GetBotName attempts to identify the specific bot name
func GetBotName(userAgent string) string {
	if userAgent == "" {
		return "Unknown Bot"
	}

	uaLower := strings.ToLower(userAgent)

	// Search engine bots
	if strings.Contains(uaLower, "googlebot") {
		return "Googlebot"
	}
	if strings.Contains(uaLower, "bingbot") {
		return "Bingbot"
	}
	if strings.Contains(uaLower, "duckduckbot") {
		return "DuckDuckBot"
	}
	if strings.Contains(uaLower, "baiduspider") {
		return "Baidu Spider"
	}
	if strings.Contains(uaLower, "yandex") {
		return "Yandex Bot"
	}
	if strings.Contains(uaLower, "slurp") {
		return "Yahoo Slurp"
	}

	// Social media bots
	if strings.Contains(uaLower, "facebookexternalhit") {
		return "Facebook Bot"
	}
	if strings.Contains(uaLower, "twitterbot") {
		return "Twitter Bot"
	}
	if strings.Contains(uaLower, "linkedinbot") {
		return "LinkedIn Bot"
	}
	if strings.Contains(uaLower, "whatsapp") {
		return "WhatsApp Bot"
	}
	if strings.Contains(uaLower, "telegrambot") {
		return "Telegram Bot"
	}

	// SEO bots
	if strings.Contains(uaLower, "ahrefsbot") {
		return "Ahrefs Bot"
	}
	if strings.Contains(uaLower, "semrushbot") {
		return "SEMrush Bot"
	}
	if strings.Contains(uaLower, "mj12bot") {
		return "Majestic Bot"
	}

	// Monitoring
	if strings.Contains(uaLower, "pingdom") {
		return "Pingdom"
	}
	if strings.Contains(uaLower, "uptimerobot") {
		return "UptimeRobot"
	}

	// HTTP libraries
	if strings.Contains(uaLower, "curl") {
		return "cURL"
	}
	if strings.Contains(uaLower, "wget") {
		return "Wget"
	}
	if strings.Contains(uaLower, "python-requests") {
		return "Python Requests"
	}
	if strings.Contains(uaLower, "go-http-client") {
		return "Go HTTP Client"
	}

	// Generic patterns
	if strings.Contains(uaLower, "bot") {
		return "Generic Bot"
	}
	if strings.Contains(uaLower, "crawler") {
		return "Crawler"
	}
	if strings.Contains(uaLower, "spider") {
		return "Spider"
	}

	return "Unknown Bot"
}
