package main

import (
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
    })
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
    log.Println("Handling index request")
    
    items, err := ListItems(1000)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "db error", http.StatusInternalServerError)
        return
    }
    
    enhancedItems := enhanceWithWASHAnalysis(items)
    stats := calculateWASHStats(items)
    
    log.Printf("Found %d items in database", len(items))
    
    tmpl, err := template.ParseFiles("templates/index.html")
    if err != nil {
        log.Printf("Error parsing template: %v", err)
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }
    
    data := map[string]interface{}{
        "Items":                    enhancedItems,
        "TotalItems":               len(items),
        "WASHItems":                stats.WASHCount,
        "InvestmentItems":          stats.InvestmentCount,
        "HasInvestmentOpportunities": stats.InvestmentCount > 0,
    }
    
    if err := tmpl.Execute(w, data); err != nil {
        log.Printf("Template execution error: %v", err)
        http.Error(w, "Template execution error", http.StatusInternalServerError)
    }
}

func handleItem(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Query().Get("id")
    if idStr == "" {
        http.Error(w, "missing id", http.StatusBadRequest)
        return
    }
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    it, err := GetItemByID(id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    enhancedItems := enhanceWithWASHAnalysis([]Item{*it})
    var enhancedItem EnhancedItem
    if len(enhancedItems) > 0 {
        enhancedItem = enhancedItems[0]
    }
    
    tmpl, err := template.ParseFiles("templates/index.html")
    if err != nil {
        log.Printf("Error parsing template: %v", err)
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }
    
    data := map[string]interface{}{
        "Items": []EnhancedItem{enhancedItem},
        "TotalItems": 1,
        "WASHItems": 1,
        "InvestmentItems": 0,
    }
    
    if err := tmpl.Execute(w, data); err != nil {
        log.Printf("template error: %v", err)
    }
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        tmpl, err := template.ParseFiles("templates/index.html")
        if err != nil {
            log.Printf("Error parsing template: %v", err)
            http.Error(w, "Template error", http.StatusInternalServerError)
            return
        }
        
        data := map[string]interface{}{
            "Items": []EnhancedItem{},
        }
        if err := tmpl.Execute(w, data); err != nil {
            log.Printf("template error: %v", err)
        }
        return
    }
    
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    title := r.FormValue("title")
    url := r.FormValue("url")
    summary := r.FormValue("summary")
    if title == "" || url == "" {
        http.Error(w, "title and url required", http.StatusBadRequest)
        return
    }
    it := &Item{
        Title:   title,
        Summary: summary,
        URL:     url,
        Source:  "manual",
    }
    if err := InsertItem(it); err != nil {
        log.Printf("Insert error: %v", err)
        http.Error(w, "db error", http.StatusInternalServerError)
        return
    }
    log.Printf("Added manual item: %s", title)
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func triggerFetchHandler(f *Fetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Println("Manual fetch triggered")
        w.Write([]byte("WASH research fetch started. Check server logs for details.\n"))
        go func() {
            if err := f.FetchAndStore(); err != nil {
                log.Printf("manual fetch error: %v", err)
            } else {
                log.Println("Manual fetch completed successfully")
            }
        }()
    }
}

func triggerArchiveHandler(w http.ResponseWriter, r *http.Request) {
    daysStr := r.URL.Query().Get("days")
    if daysStr == "" {
        daysStr = "30"
    }
    days, err := strconv.Atoi(daysStr)
    if err != nil {
        days = 30
    }
    
    log.Printf("Manual archive triggered for items older than %d days", days)
    
    go func() {
        if err := ArchiveOldItems(days, "Manual archive"); err != nil {
            log.Printf("Manual archive error: %v", err)
        } else {
            log.Printf("Manual archive completed for items older than %d days", days)
        }
    }()
    
    w.Write([]byte("Archive process started. Check server logs for details.\n"))
}

func handleArchived(w http.ResponseWriter, r *http.Request) {
    items, err := GetArchivedItems(100)
    if err != nil {
        log.Printf("Error getting archived items: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    tmpl, err := template.ParseFiles("templates/index.html")
    if err != nil {
        log.Printf("Error parsing template: %v", err)
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }
    
    data := map[string]interface{}{
        "Items":           enhanceWithWASHAnalysis(items),
        "TotalItems":      len(items),
        "WASHItems":       len(items),
        "InvestmentItems": 0,
        "IsArchivedView":  true,
    }
    
    if err := tmpl.Execute(w, data); err != nil {
        log.Printf("Template execution error: %v", err)
        http.Error(w, "Template execution error", http.StatusInternalServerError)
    }
}

// PROPOSAL HANDLERS
func handleProposals(w http.ResponseWriter, r *http.Request) {
    log.Println("Handling proposals list request")
    
    proposals, err := ListProposals()
    if err != nil {
        log.Printf("Error getting proposals: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    log.Printf("Found %d proposals", len(proposals))
    
    tmpl, err := template.ParseFiles("templates/proposals.html")
    if err != nil {
        log.Printf("Error parsing proposals template: %v", err)
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }
    
    data := map[string]interface{}{
        "Proposals": proposals,
        "ShowProposals": true,
    }
    
    if err := tmpl.Execute(w, data); err != nil {
        log.Printf("Template execution error: %v", err)
        http.Error(w, "Template execution error", http.StatusInternalServerError)
    }
}

func handleProposalView(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Query().Get("id")
    if idStr == "" {
        http.Error(w, "missing id", http.StatusBadRequest)
        return
    }
    
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    
    proposal, err := GetProposalByID(id)
    if err != nil {
        http.Error(w, "proposal not found", http.StatusNotFound)
        return
    }
    
    log.Printf("Serving proposal: %s", proposal.FilePath)
    
    // Serve the HTML file directly
    http.ServeFile(w, r, proposal.FilePath)
}

func handleProposalUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        log.Println("Showing upload form")
        
        tmpl, err := template.ParseFiles("templates/proposals.html")
        if err != nil {
            log.Printf("Error parsing upload template: %v", err)
            http.Error(w, "Template error", http.StatusInternalServerError)
            return
        }
        
        data := map[string]interface{}{
            "ShowUploadForm": true,
        }
        
        if err := tmpl.Execute(w, data); err != nil {
            log.Printf("Template execution error: %v", err)
            http.Error(w, "Template execution error", http.StatusInternalServerError)
        }
        return
    }
    
    if r.Method == http.MethodPost {
        log.Println("Processing proposal upload")
        
        if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
            http.Error(w, "File too large", http.StatusBadRequest)
            return
        }
        
        title := r.FormValue("title")
        description := r.FormValue("description")
        file, handler, err := r.FormFile("proposalFile")
        if err != nil {
            http.Error(w, "Error retrieving file", http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        if !strings.HasSuffix(handler.Filename, ".html") {
            http.Error(w, "Only HTML files allowed", http.StatusBadRequest)
            return
        }
        
        if err := os.MkdirAll("static/proposals", 0755); err != nil {
            http.Error(w, "Error creating directory", http.StatusInternalServerError)
            return
        }
        
        filePath := fmt.Sprintf("static/proposals/%d_%s", time.Now().Unix(), handler.Filename)
        dst, err := os.Create(filePath)
        if err != nil {
            http.Error(w, "Error saving file", http.StatusInternalServerError)
            return
        }
        defer dst.Close()
        
        if _, err := io.Copy(dst, file); err != nil {
            http.Error(w, "Error saving file", http.StatusInternalServerError)
            return
        }
        
        proposal := &Proposal{
            Title:       title,
            Description: description,
            FileName:    handler.Filename,
            FilePath:    filePath,
        }
        
        if err := InsertProposal(proposal); err != nil {
            http.Error(w, "Error saving proposal", http.StatusInternalServerError)
            return
        }
        
        log.Printf("Successfully uploaded proposal: %s", title)
        http.Redirect(w, r, "/proposals", http.StatusSeeOther)
    }
}

// WASH Analysis Functions
type WASHStats struct {
    WASHCount       int
    InvestmentCount int
}

func calculateWASHStats(items []Item) WASHStats {
    var stats WASHStats
    for _, item := range items {
        if isWASHRelated(item) {
            stats.WASHCount++
        }
        if isInvestmentOpportunity(item) {
            stats.InvestmentCount++
        }
    }
    return stats
}

type EnhancedItem struct {
    Item
    Tags               []string
    IsInvestment       bool
    IsInnovation       bool
    InvestmentPotential string
}

func enhanceWithWASHAnalysis(items []Item) []EnhancedItem {
    var enhanced []EnhancedItem
    for _, item := range items {
        enhancedItem := EnhancedItem{
            Item: item,
            Tags: determineWASHTags(item),
        }
        
        enhancedItem.IsInvestment = isInvestmentOpportunity(item)
        enhancedItem.IsInnovation = isInnovation(item)
        enhancedItem.InvestmentPotential = assessInvestmentPotential(item)
        
        enhanced = append(enhanced, enhancedItem)
    }
    return enhanced
}

func isWASHRelated(item Item) bool {
    washKeywords := []string{
        "water", "WASH", "sanitation", "hygiene", "purification", "filtration",
        "wastewater", "desalination", "toilet", "handwashing", "clean water",
        "water crisis", "water scarcity", "SDG6", "water infrastructure",
    }
    
    content := strings.ToLower(item.Title + " " + item.Summary)
    for _, keyword := range washKeywords {
        if strings.Contains(content, strings.ToLower(keyword)) {
            return true
        }
    }
    return false
}

func isInvestmentOpportunity(item Item) bool {
    investmentKeywords := []string{
        "funding", "investment", "startup", "venture", "capital", "series A",
        "series B", "seed round", "funding round", "seeking investment",
        "investment opportunity", "scale", "growth", "market opportunity",
    }
    
    content := strings.ToLower(item.Title + " " + item.Summary)
    for _, keyword := range investmentKeywords {
        if strings.Contains(content, strings.ToLower(keyword)) {
            return true
        }
    }
    return false
}

func isInnovation(item Item) bool {
    innovationKeywords := []string{
        "innovation", "technology", "patent", "breakthrough", "novel",
        "new technology", "invention", "research", "development",
    }
    
    content := strings.ToLower(item.Title + " " + item.Summary)
    for _, keyword := range innovationKeywords {
        if strings.Contains(content, strings.ToLower(keyword)) {
            return true
        }
    }
    return false
}

func determineWASHTags(item Item) []string {
    var tags []string
    content := strings.ToLower(item.Title + " " + item.Summary)
    
    if strings.Contains(content, "investment") || strings.Contains(content, "funding") {
        tags = append(tags, "investment")
    }
    if strings.Contains(content, "technology") || strings.Contains(content, "innovation") {
        tags = append(tags, "technology")
    }
    if strings.Contains(content, "sanitation") || strings.Contains(content, "toilet") {
        tags = append(tags, "sanitation")
    }
    if strings.Contains(content, "hygiene") || strings.Contains(content, "handwashing") {
        tags = append(tags, "hygiene")
    }
    if strings.Contains(content, "water") && (strings.Contains(content, "purification") || strings.Contains(content, "filtration")) {
        tags = append(tags, "water")
    }
    
    return tags
}

func assessInvestmentPotential(item Item) string {
    content := strings.ToLower(item.Title + " " + item.Summary)
    
    if strings.Contains(content, "series a") || strings.Contains(content, "series b") {
        return "High - Growth stage company with proven traction"
    }
    if strings.Contains(content, "seed") || strings.Contains(content, "early stage") {
        return "Medium - Early stage with market potential"
    }
    if strings.Contains(content, "breakthrough") || strings.Contains(content, "patent") {
        return "High - Protected technology with strong IP"
    }
    if strings.Contains(content, "government") || strings.Contains(content, "municipal") {
        return "Medium - Stable government contracts"
    }
    
    return ""
}