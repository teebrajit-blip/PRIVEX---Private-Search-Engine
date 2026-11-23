package api

import (
	"crypto/sha256"
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"privex/internal/analysis"
	"privex/internal/index"
	"privex/internal/query"
	"privex/internal/ranking"
)

type Server struct {
	index    *index.MemoryIndex
	ranker   *ranking.Ranker
	parser   *query.Parser
	analyzer *analysis.Analyzer

	mu         sync.Mutex
	rate       map[string][]time.Time
	cache      map[string]cachedResponse
	rateSalt   [32]byte
	rateSaltAt time.Time
}

type cachedResponse struct {
	value   searchResponse
	expires time.Time
}

type searchResponse struct {
	Results      []searchResult `json:"results"`
	TotalResults int            `json:"total_results"`
	Page         int            `json:"page"`
}

type searchResult struct {
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

func New(idx *index.MemoryIndex) *Server {
	analyzer := analysis.DefaultAnalyzer()
	return &Server{
		index:      idx,
		ranker:     ranking.NewRanker(ranking.DefaultParams()),
		parser:     query.NewParser(analyzer),
		analyzer:   analyzer,
		rate:       make(map[string][]time.Time),
		cache:      make(map[string]cachedResponse),
		rateSaltAt: time.Now(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", s.search)
	mux.HandleFunc("GET /api/suggest", s.suggest)
	mux.HandleFunc("GET /api/document/{id}", s.document)
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /healthz", s.health)
	mux.Handle("/", http.FileServer(http.Dir("web")))
	return s.headers(mux)
}

func (s *Server) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.URL.Path == "/api/search" || r.URL.Path == "/api/suggest" {
			if !s.allow(r.RemoteAddr) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	host := strings.Split(key, ":")[0]
	if host == "" {
		host = "unknown"
	}
	if now.Sub(s.rateSaltAt) >= time.Hour {
		s.rateSalt = sha256.Sum256([]byte(now.String()))
		s.rateSaltAt = now
	}
	hostKey := sha256.Sum256(append(s.rateSalt[:], []byte(host)...))
	host = string(hostKey[:])
	values := s.rate[host][:0]
	for _, t := range s.rate[host] {
		if now.Sub(t) < time.Minute {
			values = append(values, t)
		}
	}
	if len(values) >= 60 {
		s.rate[host] = values
		return false
	}
	s.rate[host] = append(values, now)
	return true
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("q")
	if len(raw) > 256 {
		http.Error(w, "query too long", http.StatusBadRequest)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 100 {
		http.Error(w, "page out of range", http.StatusBadRequest)
		return
	}
	key := strings.ToLower(strings.Join(strings.Fields(raw), " ")) + "|" + strconv.Itoa(page)
	s.mu.Lock()
	if cached, ok := s.cache[key]; ok && cached.expires.After(time.Now()) {
		s.mu.Unlock()
		writeJSON(w, cached.value)
		return
	}
	s.mu.Unlock()

	parsed := s.parser.Parse(raw)
	scored := s.ranker.Search(s.index, parsed.Terms, 20)
	start := (page - 1) * 10
	if start > len(scored) {
		start = len(scored)
	}
	end := start + 10
	if end > len(scored) {
		end = len(scored)
	}
	results := make([]searchResult, 0, end-start)
	for _, result := range scored[start:end] {
		results = append(results, searchResult{
			ID:      result.ExternalID,
			Title:   html.EscapeString(result.Document.Title),
			URL:     result.Document.URL,
			Snippet: html.EscapeString(result.Document.Snippet),
			Score:   result.Score,
		})
	}
	response := searchResponse{Results: results, TotalResults: len(scored), Page: page}
	s.mu.Lock()
	s.cache[key] = cachedResponse{value: response, expires: time.Now().Add(30 * time.Second)}
	s.mu.Unlock()
	writeJSON(w, response)
}

func (s *Server) suggest(w http.ResponseWriter, r *http.Request) {
	prefix := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(prefix) > 64 {
		http.Error(w, "query too long", http.StatusBadRequest)
		return
	}
	suggestions := make([]string, 0, 10)
	for _, term := range s.index.ListTerms(1000) {
		if strings.HasPrefix(term, prefix) {
			suggestions = append(suggestions, term)
			if len(suggestions) == 10 {
				break
			}
		}
	}
	writeJSON(w, map[string]any{"suggestions": suggestions})
}

func (s *Server) document(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	for _, scanned := range s.index.GetAllDocuments() {
		if scanned.ExternalID == id {
			writeJSON(w, map[string]any{"id": scanned.ExternalID, "title": scanned.Title, "url": scanned.URL})
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"documents": s.index.TotalDocs(), "terms": s.index.TermCount()})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"status": "ok", "documents": s.index.TotalDocs()})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
