# contacts-scraper

Sample go project scraping emails based on https://github.com/gocolly/colly

## Usage

Sample:
```
go build
./contacts-scraper -max-duration 2s -url "https://www.google.com/search?client=ubuntu&channel=fs&q=club+ciclista+espana+contacto"
```

Options:
 - `url` - start url 
 - `max-duration` - how long scraping can work as duration string
 - `max-depth` - max depth - 16 by default but domains different than the root one are blocked on level 3 