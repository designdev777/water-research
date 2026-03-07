package main

import (
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/mmcdole/gofeed"
)

type Fetcher struct {
    Sources []string
    Client  *http.Client
    Parser  *gofeed.Parser
}

func NewFetcher(sources []string) *Fetcher {
    return &Fetcher{
        Sources: sources,
        Client: &http.Client{
            Timeout: 30 * time.Second,
        },
        Parser: gofeed.NewParser(),
    }
}

func (f *Fetcher) FetchAndStore() error {
    successfulFetches := 0
    
    for _, src := range f.Sources {
        log.Printf("🔍 Fetching: %s", src)
        
        if f.isKnownBrokenFeed(src) {
            log.Printf("⏭️ Skipping known broken feed: %s", src)
            continue
        }
        
        req, err := http.NewRequest("GET", src, nil)
        if err != nil {
            log.Printf("❌ Error creating request for %s: %v", src, err)
            continue
        }
        
        f.setRequestHeaders(req, src)
        
        resp, err := f.Client.Do(req)
        if err != nil {
            log.Printf("❌ Network error fetching %s: %v", src, err)
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
            log.Printf("⚠️ Non-200 status for %s: %d - %s", src, resp.StatusCode, resp.Status)
            f.markFeedAsBroken(src)
            continue
        }

        feed, err := f.Parser.Parse(resp.Body)
        if err != nil {
            log.Printf("❌ Error parsing RSS feed %s: %v", src, err)
            continue
        }

        log.Printf("✅ Successfully parsed: %s (%d items)", feed.Title, len(feed.Items))
        
        itemsStored := 0
        for _, item := range feed.Items {
            if err := f.processFeedItem(item, src, feed.Title); err != nil {
                log.Printf("⚠️ Error processing item: %v", err)
            } else {
                itemsStored++
            }
        }
        
        log.Printf("📥 Stored %d items from %s", itemsStored, feed.Title)
        successfulFetches++
        
        time.Sleep(2 * time.Second)
    }
    
    log.Printf("🎯 Fetch completed: %d/%d sources successful", successfulFetches, len(f.Sources))
    return nil
}

func (f *Fetcher) isKnownBrokenFeed(url string) bool {
    brokenFeeds := []string{
        "https://www.wri.org/insights/rss",
        "https://www.watertechonline.com/feed/",
    }
    
    for _, broken := range brokenFeeds {
        if url == broken {
            return true
        }
    }
    return false
}

func (f *Fetcher) markFeedAsBroken(url string) {
    log.Printf("📝 Marking as broken feed: %s", url)
}

func (f *Fetcher) setRequestHeaders(req *http.Request, url string) {
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WaterResearchBot/1.0; +research@allenwater.co.uk)")
    
    if strings.Contains(url, "arxiv.org") {
        req.Header.Set("Accept", "application/rss+xml, application/atom+xml")
    } else if strings.Contains(url, "mdpi.com") || strings.Contains(url, "frontiersin.org") {
        req.Header.Set("Accept", "application/rss+xml, application/xml")
    } else {
        req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")
    }
}

func (f *Fetcher) processFeedItem(item *gofeed.Item, sourceURL string, sourceName string) error {
    if item.Title == "" || item.Link == "" {
        return nil
    }

    title := strings.ToLower(item.Title)
    description := strings.ToLower(item.Description)
    
    waterKeywords := []string{
        "water", "WASH", "sanitation", "hygiene", "purification", "filtration",
        "wastewater", "desalination", "toilet", "handwashing", "clean water",
        "water crisis", "water scarcity", "SDG6", "water infrastructure",
        "hydrology", "aquifer", "irrigation", "water treatment", "sewage",
        "water quality", "drinking water", "water management", "conservation",
    }
    
    hasWaterKeyword := false
    for _, keyword := range waterKeywords {
        if strings.Contains(title, keyword) || strings.Contains(description, keyword) {
            hasWaterKeyword = true
            break
        }
    }
    
    if !hasWaterKeyword {
        return nil
    }
    
    if strings.Contains(sourceURL, "arxiv.org") {
        sourceName = "arXiv Preprints - " + sourceName
    } else if strings.Contains(sourceURL, "mdpi.com") {
        sourceName = "MDPI Journals - " + sourceName
    } else if strings.Contains(sourceURL, "frontiersin.org") {
        sourceName = "Frontiers Journals - " + sourceName
    }

    dbItem := &Item{
        Title:   item.Title,
        Summary: item.Description,
        URL:     item.Link,
        Source:  sourceName,
    }

    if item.PublishedParsed != nil {
        dbItem.PublishedAt.Valid = true
        dbItem.PublishedAt.Time = *item.PublishedParsed
    } else if item.UpdatedParsed != nil {
        dbItem.PublishedAt.Valid = true
        dbItem.PublishedAt.Time = *item.UpdatedParsed
    }

    if err := InsertItem(dbItem); err != nil {
        if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
            return err
        }
    } else {
        log.Printf("➕ Stored: %s", item.Title)
    }
    
    return nil
}