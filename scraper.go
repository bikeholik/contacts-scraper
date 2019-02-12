package main

import (
	"flag"
	"github.com/deckarep/golang-set"
	"github.com/gocolly/colly"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func scrape(startUrl string, maxDuration time.Duration, emails chan string) {
	// TODO move outside
	r := regexp.MustCompile("[\\w]+@[\\w]+\\.[\\w]+(\\.[\\w]+)?")
	limitTime := time.Now().Add(maxDuration)
	log.Println("Scraping will work until", limitTime)

	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 5.1; rv:7.0.1) Gecko/20100101 Firefox/7.0.1"),
		colly.Async(true),
		colly.DisallowedDomains("www.facebook.com", "twitter.com"),
		colly.DisallowedURLFilters(regexp.MustCompile(".*facebook.com")),
		colly.CacheDir("./.cache"),
		colly.MaxDepth(8),
	)

	_ = collector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})

	collector.OnResponse(func(response *colly.Response) {
		log.Println("Parsing", response.Request.URL)
		all := r.FindAll(response.Body, -1)
		for _, s := range all {
			emails <- string(s)
		}
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if time.Now().Before(limitTime) {
			link := e.Request.AbsoluteURL(e.Attr("href"))
			if strings.HasPrefix(link, "http") {
				err := e.Request.Visit(link)
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
	})

	err := collector.Visit(startUrl)

	if err != nil {
		log.Fatalln("Visiting", startUrl, "failed:", err)
	}

	collector.Wait()

	close(emails)
}

func main() {

	optStartUrl := flag.String("url", "http://dodekstudio.com/contact-me.php?lang=es", "url from which to start")
	optMaxDuration := flag.String("max-duration", "30s", "max scraping duration")

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

	ch := make(chan string)

	go scrape(u.String(), maxDuration, ch)

	emails := mapset.NewSet()

	for email := range ch {
		if emails.Add(email) {
			println("Found: ", email)
		}
	}
}
