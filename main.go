package main

import (
    "fmt"
    "net/http"
    "golang.org/x/net/html"
	"regexp"
	"strings"
)

type PageKeywords struct {
    Title           string
    Headings        []string
    ContentKeywords []string
}

type QueueItem struct {
	URL   string
	Depth int
}

var StopWords = map[string]struct{}{
    "a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {}, "if": {}, "while": {}, "with": {}, "of": {},
    "at": {}, "by": {}, "for": {}, "to": {}, "in": {}, "on": {}, "up": {}, "out": {}, "over": {}, "under": {},
    "as": {}, "is": {}, "it": {}, "this": {}, "that": {}, "these": {}, "those": {}, "he": {}, "she": {}, "they": {},
    "we": {}, "you": {}, "i": {}, "me": {}, "him": {}, "her": {}, "them": {}, "us": {}, "my": {}, "your": {},

    "un": {}, "una": {}, "unos": {}, "unas": {}, "el": {}, "la": {}, "los": {}, "las": {}, "y": {}, "o": {},
    "pero": {}, "si": {}, "mientras": {}, "con": {}, "de": {}, "en": {}, "sobre": {}, "bajo": {}, "por": {},
    "es": {}, "esto": {}, "eso": {}, "estos": {}, "esas": {}, "él": {}, "ella": {}, "ellos": {}, "nosotros": {},
    "tú": {}, "yo": {}, "te": {}, "le": {}, "nos": {},

    "une": {}, "des": {}, "les": {}, "et": {}, "ou": {}, "mais": {}, "pendant": {}, "avec": {}, "à": {}, "dans": {},
    "sur": {}, "sous": {}, "pour": {}, "est": {}, "ce": {}, "cette": {}, "ces": {}, "il": {}, "elle": {}, "ils": {},
    "nous": {}, "vous": {}, "je": {},
}

func main() {
	seed := "https://es.wikipedia.org/wiki/Wikipedia:Portada"
	maxDepth := 1
    queue, err := buildCrawlGraph(seed, maxDepth)
	if err != nil {
		fmt.Printf("Error in getting queue for crawling: %v\n", err)
	} else {
		fmt.Println(queue)
	}

	page, err := extractKeywordsFromURL(seed)
    if err != nil {
        fmt.Printf("Error in extracting keywords from %s: %v\n", seed, err)
    }
	fmt.Println("Title:", page.Title)
	fmt.Println("Headings:", page.Headings)
	fmt.Println("Content keywords (first 50):", page.ContentKeywords[:10])
	
}

func buildCrawlGraph(seed string, maxDepth int) ([]string, error) {
    urlQueue := []QueueItem{{URL: seed, Depth: 0}}
    visited := make(map[string]bool)
    result := []string{} 

    for len(urlQueue) > 0 {
        currentItem := urlQueue[0]
        urlQueue = urlQueue[1:]

        currentURL := currentItem.URL
        currentDepth := currentItem.Depth

        if visited[currentURL] {
            continue
        }
        visited[currentURL] = true

        result = append(result, currentURL)

        if currentDepth >= maxDepth {
            continue
        }

        links, err := parseLinksFromUrl(currentURL)
        if err != nil {
            fmt.Printf("Error getting links from %s: %v\n", currentURL, err)
            continue
        }

        for _, link := range links {
            if !visited[link] {
                urlQueue = append(urlQueue, QueueItem{URL: link, Depth: currentDepth + 1})
            }
        }
    }

    return result, nil
}



func parseLinksFromUrl(url string) ([]string, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoCrawler/1.0)")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    doc, err := html.Parse(resp.Body)
    if err != nil {
        return nil, err
    }

    var links []string
    var f func(*html.Node)
    f = func(n *html.Node) {
        if n.Type == html.ElementNode && n.Data == "a" {
            for _, a := range n.Attr {
                if a.Key == "href" && isHttpsUrl(a.Val) {
                    links = append(links, a.Val)
                }
            }
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            f(c)
        }
    }
    f(doc)
    return links, nil
}

func isHttpsUrl(url string) bool {
    pattern := `^https://[^\s]+$`
    matched, _ := regexp.MatchString(pattern, url)
    return matched
}

func extractText(n *html.Node, skipTags map[string]struct{}) string {
    if n.Type == html.TextNode {
        return n.Data
    }
    if n.Type == html.ElementNode {
        if _, skip := skipTags[n.Data]; skip {
            return ""
        }
    }
    var result string
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        result += extractText(c, skipTags) + " "
    }
    return strings.Join(strings.Fields(result), " ")
}

func extractKeywordsFromURL(url string) (*PageKeywords, error) {
    client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyCrawler/1.0; +https://example.com/crawler)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

    doc, err := html.Parse(resp.Body)
    if err != nil {
        return nil, err
    }

    result := &PageKeywords{}
    var f func(*html.Node)
    skipTags := map[string]struct{}{"script": {}, "style": {}}

    f = func(n *html.Node) {
        if n.Type == html.ElementNode {
            // Extract title
            if n.Data == "title" && n.FirstChild != nil {
                result.Title = n.FirstChild.Data
            }
            // Extract h1 and h2 headings
            if n.Data == "h1" || n.Data == "h2" {
                headingText := extractText(n, skipTags)
                if headingText != "" {
                    result.Headings = append(result.Headings, headingText)
                }
            }
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            f(c)
        }
    }
    f(doc)

    fullText := extractText(doc, skipTags)
    words := strings.Fields(fullText)
    for _, w := range words {
        w = strings.ToLower(w)
        if _, isStopword := StopWords[w]; !isStopword {
            result.ContentKeywords = append(result.ContentKeywords, w)
        }
    }

    return result, nil
}

