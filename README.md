# PRIVEX — Independent, Privacy-First Search Engine

PRIVEX is an independent search engine written from scratch in Go. It operates on a **two-plane architecture** (offline ingestion and online query serving) connected through immutable on-disk inverted index segments, powered by **Okapi BM25** ranking.

---

## Architecture Overview

```
INGESTION PLANE (Offline / Batch)
Seeds/Sitemaps --> URL FRONTIER --> FETCHER --> EXTRACTOR --> ANALYZER
                   (polite queues)  (robots,    (clean text,   (tokens,
                                     SSRF guard) links)         positions)
                                         |          |                |
                                         v          v                v
                                     PostgreSQL (urls, docs) -> INDEX BUILDER
                                                                     |
                                                                     v
                                                            INDEX SEGMENTS (disk)
                                                                     |
SERVING PLANE (Online / Read-Only / Fast)                            v
User --> HTTPS --> Search API --> Query Parser -----------> Searcher & Ranker
                                  (shared analyzer)         (BM25 + Top-K Heap)
                                                                     |
                                                             Snippet Generator
                                                             (PostgreSQL text)
                                                                     |
                                                            JSON / Embedded UI
```

---

## Architectural Invariants & Corrections

1. **Two Separate Planes**: Search queries never wait on crawlers; query nodes only read immutable segments without locks.
2. **Positional Inverted Index**: All tokens are indexed with their 1-based sequential positions. Stop words are **never removed** at index time to prevent breaking phrases ("to be or not to be") and offset calculations.
3. **BM25 Non-Zero IDF**: Standard Robertson-Spärck Jones formulation:
   $$\text{idf}(t) = \ln\left(1 + \frac{N - n_t + 0.5}{n_t + 0.5}\right)$$
   guaranteed strictly positive for any document frequency.
4. **Privacy as Architecture**:
   - Zero query string recording in access logs or persistent storage.
   - Strict `Referrer-Policy: no-referrer` and `rel="noreferrer noopener"` on all outbound result links.
   - Zero external CDNs, tracking pixels, or third-party cloud AI APIs.
   - Suggestions generated exclusively from indexed document vocabulary and headings (Trie/FST).
5. **Polite, Secure Crawler**:
   - Post-DNS IP validation guarding against SSRF (blocking loopback, RFC1918, link-local, and cloud metadata ranges).
   - Per-host token bucket politeness queues respecting `robots.txt` and `Crawl-delay`.
   - `SELECT ... FOR UPDATE SKIP LOCKED` lease claiming for crash resilience.

---

## Project Structure

```
.
├── cmd/
│   ├── privex-cli/      # Milestone 1 Interactive Search CLI
│   ├── crawler/         # Ingestion worker & URL frontier (M4)
│   ├── indexer/         # Segment builder & LSM merger (M3/M5)
│   ├── server/          # HTTP Serving plane & REST API (M6)
│   └── evaltool/        # Evaluation harness for nDCG@10 (M2)
├── internal/
│   ├── analysis/        # Unicode tokenizer, normalizer, Porter stemmer
│   ├── index/           # Inverted index buffer, segments, postings
│   ├── ranking/         # Robertson-Spärck Jones BM25 & Top-K Min-Heap
│   ├── query/           # Query parser (terms, phrases, booleans)
│   ├── snippet/         # Context window extractor & XSS-safe highlighter
│   ├── frontier/        # Priority & per-host politeness queues
│   ├── fetcher/         # SSRF-guarded HTTP client & robots.txt parser
│   ├── extractor/       # HTML content density extractor & link collector
│   ├── urlnorm/         # Conservative URL normalizer
│   ├── store/           # PostgreSQL persistence via pgx
│   └── api/             # HTTP handlers with query-scrubbed logging
├── migrations/          # PostgreSQL schema migrations
├── testdata/            # Sample corpus and judged test queries
└── docs/                # Architecture Decision Records (ADRs)
```

---

## Getting Started

### 1. Prerequisites
- **Go 1.22+**
- **PostgreSQL 16+** (for crawler and document store)

### 2. Running Unit Tests
```bash
go test -v ./internal/...
```

### 3. Running the Milestone 1 CLI
```bash
# Run interactive search over the sample corpus
go run ./cmd/privex-cli

# Run a single query
go run ./cmd/privex-cli -query "privacy architecture"
go run ./cmd/privex-cli -query "\"bm25 ranking\""
```
