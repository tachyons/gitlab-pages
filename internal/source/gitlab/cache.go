package gitlab

import (
	"context"
	"fmt"
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	shortCache *cache.Cache
	longCache  *cache.Cache
	lockCache  *sync.Map
}

type lock struct {
	time time.Time
	once *sync.Once
	cond *sync.Cond
}

type retrieveFunc func(ctx context.Context) (Lookup, error)

var (
	maxRetrievalInterval = time.Second
	maxRetrievalTimeout  = 4 * time.Second
)

// NewCache creates a new instance of Cache and sets default expiration.
// Short cache expiration - 5 seconds
// Long cache expiration - 10 minutes
// Cache cleanup interval - 1 minute
func NewCache() *Cache {
	return &Cache{
		shortCache: cache.New(5*time.Second, time.Minute),
		longCache:  cache.New(10*time.Minute, time.Minute),
		lockCache:  &sync.Map{},
	}
}

// GetLookup is going to return a Lookup based on a domain name using following
// algorithm:
// - if a domain lookup is present in the short cache it will return just it
// - if it is not present in a short cache it will check the long cache
// - if it is present in a long cache it will return the long cache version and
//   run an update in a separate thread that will fetch the lookup from the
//   GitLab source and replace the short and long cache entries
// - if a domain lookup is not present in the long cache we will fetch the
//   lookup from the domain source and client will need to wait
// TODO add error handling to Lookup
func (c *Cache) GetLookup(domain string, retriever retrieveFunc) *Lookup {
	// return lookup if it exists in the short cache
	if lookup, exists := c.shortCache.Get(domain); exists {
		return lookup.(*Lookup)
	}

	// return lookup it if exists in the long cache, schedule retrieval
	if lookup, exists := c.longCache.Get(domain); exists {
		c.withLock(domain, func(lock *lock) {
			lock.once.Do(func() { go c.retrieve(domain, retriever) })
		})

		return lookup.(*Lookup)
	}

	// perform retrieval once and wait for the response
	c.withLock(domain, func(lock *lock) {
		lock.once.Do(func() { go c.retrieve(domain, retriever) })
		lock.cond.Wait()
	})

	fmt.Println("GetLookup")
	return c.GetLookup(domain, retriever)
}

func (c *Cache) retrieve(domain string, retriever retrieveFunc) {
	var lookup Lookup
	response := make(chan Lookup)
	defer close(response)

	ctx, cancel := context.WithTimeout(context.Background(), maxRetrievalTimeout)
	defer cancel()

	go c.retrievalLoop(ctx, retriever, response)

	select {
	case <-ctx.Done():
		lookup = Lookup{} // TODO store error
	case lookup = <-response:
		fmt.Println("response received") // TODO Log response
	}

	c.withLock(domain, func(lock *lock) {
		c.store(domain, lookup)
		c.lockCache.Delete(domain)
		lock.cond.Broadcast() // broadcast lookup message to all listeners
	})
}

func (c *Cache) retrievalLoop(ctx context.Context, retriever retrieveFunc, response chan<- Lookup) {
	for {
		if ctx.Err() != nil {
			return
		}

		lookup, err := retriever(ctx)

		if err != nil {
			time.Sleep(maxRetrievalInterval)
		} else {
			response <- lookup
			return
		}
	}
}

func (c *Cache) withLock(domain string, block func(*lock)) {
	newLock := &lock{
		time: time.Now(),
		once: &sync.Once{},
		cond: sync.NewCond(&sync.Mutex{}),
	}

	// go-cache does not have atomic load and store
	storedLock, _ := c.lockCache.LoadOrStore(domain, newLock)
	cacheLock := storedLock.(*lock)

	if cacheLock.isExpired() { // custom lock expiration
		c.lockCache.Delete(domain) // remove expired lock
		c.withLock(domain, block)  // retry aquiring lock
		return
	}

	cacheLock.cond.L.Lock()
	block(cacheLock)
	cacheLock.cond.L.Unlock()
}

func (c *Cache) store(domain string, lookup Lookup) *Lookup {
	c.shortCache.SetDefault(domain, &lookup)
	c.longCache.SetDefault(domain, &lookup)

	return &lookup
}

func (l *lock) isExpired() bool {
	return l.time.Add(10 * time.Second).Before(time.Now())
}
