package cache

import "time"

var (
	shortCacheExpiry     = 30 * time.Second
	longCacheExpiry      = 10 * time.Minute
	retrievalTimeout     = 5 * time.Second
	maxRetrievalInterval = time.Second
	maxRetrievalRetries  = 3
)
