package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "github.com/robfig/cron/v3"
)

func main() {
    // initialize DB (creates file research.db in cwd)
    if err := InitDB("research.db"); err != nil {
        log.Fatalf("InitDB error: %v", err)
    }
    defer DB.Close()

    // Initialize proposals table
    if err := CreateProposalsTable(); err != nil {
        log.Printf("Error creating proposals table: %v", err)
    }

    // Create fetcher with WASH-focused RSS feeds
 /*   f := NewFetcher([]string{
        "https://www.sciencedaily.com/rss/earth_climate/water.xml",
        "https://www.waterworld.com/rss",
        "https://www.wateronline.com/feed",
        "https://www.mdpi.com/search/rss?q=water",
        "https://www.frontiersin.org/journals/water/rss",
        "https://feeds.feedburner.com/IWMI-publications",
        "https://feeds.feedburner.com/iwmi-cgspace-ja",
    })
    */
	f := NewFetcher([]string{
    // Core Water Research
    "https://www.sciencedaily.com/rss/earth_climate/water.xml",
    "https://www.waterworld.com/rss",
    "https://www.wateronline.com/feed",    
    
    // Open Access Scientific Journals
    "https://www.mdpi.com/search/rss?q=water", // MDPI Water journals
    "https://www.mdpi.com/search/rss?q=WASH",
    "https://www.mdpi.com/search/rss?q=wastewater",
    "https://www.frontiersin.org/journals/water/rss",
    "https://www.frontiersin.org/journals/environmental-science/rss",
    "https://feeds.feedburner.com/IWMI-publications",
    //"https://feeds.feedburner.com/iwmi-cgspace-ja",
    
    // Preprint Servers (cutting-edge research)
    "https://export.arxiv.org/rss/cs.CE", // Computational Engineering (water modeling)
    "https://export.arxiv.org/rss/physics.chem-ph", // Chemical Physics (water treatment)
    
    // Research Institutions
    "https://www.iwapublishing.com/rss/news",
    "https://www.iwra.org/rss",
    "https://www.un-ihe.org/rss.xml",
    "https://www.nature.com/natwater.rss",
    "https://journals.plos.org/water/feed/atom",
    "https://www.ncbi.nlm.nih.gov/pmc/journals/water/",
    
    // Government & NGO Research
    "https://www.epa.gov/newsreleases/search/rss/all/rss.xml",
    "https://www.usgs.gov/news/rss/topic/water",
    "https://www.unwater.org/rss/news",
})
    // run an initial fetch at startup
    if err := f.FetchAndStore(); err != nil {
        log.Printf("initial fetch error: %v", err)
    }

    // Start cron for scheduled tasks
    c := cron.New(cron.WithChain())
    
    // Daily fetch at 02:00
    _, err := c.AddFunc("0 2 * * *", func() {
        log.Println("[cron] running daily fetch")
        if err := f.FetchAndStore(); err != nil {
            log.Printf("[cron] fetch error: %v", err)
        }
    })
    if err != nil {
        log.Fatalf("cron add error: %v", err)
    }
    
    // Archive old items (older than 30 days) every Sunday at 03:00
    _, err = c.AddFunc("0 3 * * 0", func() {
        log.Println("[cron] archiving old items")
        if err := ArchiveOldItems(30, "Auto-archive: older than 30 days"); err != nil {
            log.Printf("[cron] archive error: %v", err)
        } else {
            log.Println("[cron] archive completed")
        }
    })
    if err != nil {
        log.Fatalf("cron archive add error: %v", err)
    }
    
    c.Start()
    defer c.Stop()

    // Setup HTTP server and handlers
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleIndex)
    mux.HandleFunc("/item", handleItem)
    mux.HandleFunc("/add", handleAdd)
    mux.HandleFunc("/trigger-fetch", triggerFetchHandler(f))
    mux.HandleFunc("/trigger-archive", triggerArchiveHandler)
    mux.HandleFunc("/archived", handleArchived)
    mux.HandleFunc("/proposals", handleProposals)
    mux.HandleFunc("/proposal", handleProposalView)
    mux.HandleFunc("/upload-proposal", handleProposalUpload)

    // Add static file serving for proposal files
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

// Get PORT from environment (Render sets this automatically)
port := os.Getenv("PORT")
if port == "" {
    port = "8080" // fallback for local development
}

srv := &http.Server{
    Addr:         ":" + port,  // Now uses Render's dynamic port
    Handler:      loggingMiddleware(mux),
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}

    // graceful shutdown on Ctrl+C
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt)

    go func() {
        log.Println("server starting on http://localhost:8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %v", err)
        }
    }()

    <-stop
    log.Println("shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server shutdown: %v", err)
    }
    log.Println("server stopped")
}
