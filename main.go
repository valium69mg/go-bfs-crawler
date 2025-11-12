package main

import (
    "fmt"
    "net/http"
    "golang.org/x/net/html"
	"regexp"
)

func main() {
	seed := "https://es.wikipedia.org/wiki/Wikipedia:Portada"
	maxDepth := 1
    queue, err := buildCrawlGraph(seed, maxDepth)
	if err != nil {
		fmt.Printf("Error in getting queue for crawling: %v\n", err)
	} else {
		fmt.Println(queue)
	}
	
}

func buildCrawlGraph(seed string, maxDepth int) ([]string, error) {
    type QueueItem struct {
        URL   string
        Depth int
    }

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


