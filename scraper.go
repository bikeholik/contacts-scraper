package main

import (
	"flag"
	"fmt"
	"github.com/deckarep/golang-set"
	"github.com/gocolly/colly"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type FoundEmail struct {
	email     string
	sourceUrl string
}

const RFC_MAIL_REGEXP = "(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\\])"
const SIMPLE_MAIL_REGEXP = "[\\w]+@[\\w]+\\.[\\w]+(\\.[\\w]+)?"
const MAIL_REGEXP = SIMPLE_MAIL_REGEXP

var DISALLOWED_EXTENSIONS = []string{".png", ".gif", ".jpg", ".jpeg"}

func scrape(startUrl *url.URL, maxDepth int, maxDuration time.Duration, emails chan FoundEmail) {

	mailRegexp := regexp.MustCompile(MAIL_REGEXP)
	limitTime := time.Now().Add(maxDuration)

	log.Println("Scraping will work until", limitTime)

	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 5.1; rv:7.0.1) Gecko/20100101 Firefox/7.0.1"),
		colly.Async(true),
		colly.DisallowedDomains("www.facebook.com", "twitter.com"),
		colly.DisallowedURLFilters(regexp.MustCompile(".*facebook.com")),
		colly.CacheDir("./.cache"),
		colly.MaxDepth(maxDepth),
	)

	_ = collector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})

	collector.OnResponse(func(response *colly.Response) {
		log.Println("Parsing", response.Request.URL)
		all := mailRegexp.FindAll(response.Body, -1)
		for _, s := range all {
			emails <- FoundEmail{email: string(s), sourceUrl: response.Request.URL.String()}
		}
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if time.Now().Before(limitTime) {
			link := e.Attr("href")
			if strings.HasPrefix(link, "http") {
				r := e.Request
				err := r.Visit(link)
				if err != nil {
					log.Println("Visiting", link, "failed:", err)
				}
			}
		} else {
			log.Println("Time limit reached")
		}
	})

	collector.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
		if r.URL.Host != startUrl.Host && r.Depth > 2 {
			collector.DisallowedURLFilters = append(collector.DisallowedURLFilters, regexp.MustCompile(".*"+r.URL.Host))
		}
	})

	err := collector.Visit(startUrl.String())

	if err != nil {
		log.Fatalln("Visiting", startUrl.String(), "failed:", err)
	}

	collector.Wait()

	close(emails)
}

func shouldBeIgnored(email string) bool {
	for _, s := range DISALLOWED_EXTENSIONS {
		if s == email {
			return true
		}
	}
	return false
}

func main() {

	f, _ := os.OpenFile("scraper.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	log.SetOutput(f)

	optStartUrl := flag.String("url", "http://dodekstudio.com/contact-me.php?lang=es", "url from which to start")
	optMaxDuration := flag.String("max-duration", "30s", "max scraping duration")
	optDepth := flag.Int("max-depth", 16, "max depth on main page")

	flag.Parse()

	if (*optStartUrl) == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	u, err := url.Parse(*optStartUrl)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	maxDuration, err := time.ParseDuration(*optMaxDuration)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	ch := make(chan FoundEmail)

	go scrape(u, *optDepth, maxDuration, ch)

	emails := mapset.NewSet()

	for email := range ch {
		if emails.Add(email.email) && !shouldBeIgnored(email.email) {
			fmt.Printf("%s at %s\n", email.email, email.sourceUrl)
		}
	}
}
