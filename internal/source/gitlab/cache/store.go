package cache

import "context"

// Store defines an interface describing an abstract cache store
type Store interface {
	LoadOrCreate(ctx context.Context, domain string) *Entry
	ReplaceOrCreate(ctx context.Context, domain string) *Entry
}
