package internal

type CorrIDOriginalURL struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}

type CorrIDUrlID struct {
	CorrID string
	URLID  int
}

type IDToDelete struct {
	ID     int
	UserID int
}
