package cache

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
}

// NewCache creates a new instance of Cache and sets default expiration
func NewCache() *Cache {
	return &Cache{}
}
