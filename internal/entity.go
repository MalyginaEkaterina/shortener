package internal

// CorrIDOriginalURL contains original URL and its correlation_id.
type CorrIDOriginalURL struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}

// CorrIDUrlID contains shortened url ID and associated correlation_id.
type CorrIDUrlID struct {
	CorrID string
	URLID  int
}

// IDToDelete contains shortened url ID for deletion and user ID from the request.
type IDToDelete struct {
	ID     int
	UserID int
}
