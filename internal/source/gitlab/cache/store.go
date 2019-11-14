package cache

// Store defines an interface describing an abstract cache store
type Store interface {
	LoadOrCreate(domain string) *Entry
	ReplaceOrCreate(domain string, entry *Entry) *Entry
}
