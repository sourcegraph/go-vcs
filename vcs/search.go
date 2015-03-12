package vcs

type Searcher interface {
	// Search searches the text of a repository at the given commit
	// ID.
	Search(CommitID, SearchOptions) ([]*SearchResult, error)
}

// SearchOptions specifies options for a repository search.
type SearchOptions struct {
	Query        string // the query string
	QueryType    string // currently only FixedQuery ("fixed") is supported
	ContextLines int    // the number of lines before and after each hit to display

	N      int // max number of matches to return
	Offset int // starting offset for matches (use with N for pagination)
}

const (
	// FixedQuery is a value for SearchOptions.QueryType that
	// indicates the query is a fixed string, not a regex.
	FixedQuery = "fixed"

	// TODO(sqs): allow regexp searches, extended regexp searches, etc.
)

// A SearchResult is a match returned by a search.
type SearchResult struct {
	File string // the file that contains this match

	StartByte, EndByte uint32 // the byte range [start,end) of the match

	StartLine, EndLine uint32 // the line range [start,end] of the match

	// Match is the matching portion of the file from [StartByte,
	// EndByte).
	Match []byte
}
