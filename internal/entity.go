package internal

type CorrIDOriginalURL struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}

type CorrIDShortURL struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

type CorrIDUrlID struct {
	CorrID string
	URLID  int
}
