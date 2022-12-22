package storage

type Storage struct {
	Urls []string
}

func (s *Storage) AddURL(url string) int {
	s.Urls = append(s.Urls, url)
	return len(s.Urls) - 1
}

func (s *Storage) ValidID(id int) bool {
	return id >= 0 && id < len(s.Urls)
}

func (s *Storage) GetURL(id int) string {
	return s.Urls[id]
}
