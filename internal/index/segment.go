package index

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
)

type SegmentHeader struct {
	Version       uint32 `json:"version"`
	DocCount      uint32 `json:"doc_count"`
	Terms         uint32 `json:"terms"`
	PostingsBytes uint64 `json:"postings_bytes"`
}

type segmentPostings struct {
	DocID     uint32   `json:"doc_id"`
	Positions []uint32 `json:"positions"`
}

type segmentData struct {
	Header    SegmentHeader                `json:"header"`
	Documents []Document                   `json:"documents"`
	Postings  map[string][]segmentPostings `json:"postings"`
}

// WriteSegment writes a deterministic JSON segment. The format is migration-
// friendly and can be replaced by varint postings later without changing this API.
func (idx *MemoryIndex) WriteSegment(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	docs := make([]Document, 0, len(idx.docs))
	postings := make(map[string][]segmentPostings, len(idx.postings))
	for term, list := range idx.postings {
		out := make([]segmentPostings, 0, len(list))
		for _, posting := range list {
			out = append(out, segmentPostings{DocID: posting.DocID, Positions: append([]uint32(nil), posting.Positions...)})
		}
		postings[term] = out
	}
	for id, doc := range idx.docs {
		_ = id
		docs = append(docs, doc)
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].DocID < docs[j].DocID })
	data := segmentData{Header: SegmentHeader{Version: 1, DocCount: uint32(len(docs)), Terms: uint32(len(postings))}, Documents: docs, Postings: postings}
	return writeJSONAtomically(path, data)
}

func writeJSONAtomically(path string, data segmentData) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ReadSegment(path string) (*MemoryIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data segmentData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	if data.Header.Version != 1 {
		return nil, errors.New("unsupported segment version")
	}
	idx := NewMemoryIndex()
	for _, doc := range data.Documents {
		termPositions := make(map[string][]uint32)
		for term, list := range data.Postings {
			for _, posting := range list {
				if posting.DocID == doc.DocID {
					termPositions[term] = append(termPositions[term], posting.Positions...)
				}
			}
		}
		idx.AddDocument(doc, termPositions)
	}
	return idx, nil
}

func CopySegment(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
