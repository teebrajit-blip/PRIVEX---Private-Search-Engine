package frontier

import (
	"container/heap"
	"sync"
	"time"
)

type Item struct {
	URL       string
	Priority  int
	NextFetch time.Time
	Host      string
	Depth     int
	index     int
}

type Queue []*Item

func (q Queue) Len() int { return len(q) }
func (q Queue) Less(i, j int) bool {
	if q[i].Priority == q[j].Priority {
		return q[i].NextFetch.Before(q[j].NextFetch)
	}
	return q[i].Priority > q[j].Priority
}
func (q Queue) Swap(i, j int) { q[i], q[j] = q[j], q[i]; q[i].index = i; q[j].index = j }
func (q *Queue) Push(x any)   { n := len(*q); item := x.(*Item); item.index = n; *q = append(*q, item) }
func (q *Queue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*q = old[:n-1]
	return item
}

type Frontier struct {
	mu    sync.Mutex
	queue Queue
	seen  map[string]struct{}
}

func New() *Frontier { q := &Frontier{seen: make(map[string]struct{})}; heap.Init(&q.queue); return q }

func (f *Frontier) Add(rawURL, host string, priority, depth int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.seen[rawURL]; exists {
		return false
	}
	f.seen[rawURL] = struct{}{}
	heap.Push(&f.queue, &Item{URL: rawURL, Host: host, Priority: priority, Depth: depth, NextFetch: time.Now()})
	return true
}

func (f *Frontier) Claim() (*Item, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.queue.Len() == 0 {
		return nil, false
	}
	item := f.queue[0]
	if item.NextFetch.After(time.Now()) {
		return nil, false
	}
	heap.Pop(&f.queue)
	return item, true
}

func (f *Frontier) Requeue(item *Item, delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item.NextFetch = time.Now().Add(delay)
	heap.Push(&f.queue, item)
}
