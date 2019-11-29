package cache

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
}

// New creates a new instance of Cache and sets default expiration
func New() *Cache {
	return &Cache{}
}
