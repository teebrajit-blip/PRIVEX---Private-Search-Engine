package index

// Posting holds the inverted index entry for a term within a document.
// Positions are 1-based sequential token positions.
// Term Frequency (tf) is len(Positions).
type Posting struct {
	DocID     uint32
	Positions []uint32
}

// FieldLengths stores the token count per document field.
type FieldLengths struct {
	Title    uint32
	Headings uint32
	Body     uint32
	URL      uint32
	Anchor   uint32
	Total    uint32
}

// Document represents an indexed document and its stored metadata.
type Document struct {
	DocID       uint32
	ExternalID  int64 // Maps to PostgreSQL BIGINT
	URL         string
	Title       string
	Snippet     string
	Lengths     FieldLengths
	TotalLength uint32
}
