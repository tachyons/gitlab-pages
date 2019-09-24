package source

import (
	"strings"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/groups"
)

// Domains struct represents a map of all domains supported by pages. It is
// currently reading them from disk.
type Domains struct {
	dm   groups.Map
	lock sync.RWMutex
}

// GetDomain returns a domain from the domains map
func (d *Domains) GetDomain(host string) *domain.Domain {
	host = strings.ToLower(host)
	d.lock.RLock()
	defer d.lock.RUnlock()
	domain, _ := d.dm[host]

	return domain
}

// HasDomain checks for presence of a domain in the domains map
func (d *Domains) HasDomain(host string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()

	host = strings.ToLower(host)
	_, isPresent := d.dm[host]

	return isPresent
}

// Ready checks if the domains source is ready for work
func (d *Domains) Ready() bool {
	return d.dm != nil
}

// Watch starts the domain source, in this case it is reading domains from
// groups on disk concurrently
func (d *Domains) Watch(rootDomain string) {
	go groups.Watch(rootDomain, d.updateDomains, time.Second)
}

func (d *Domains) updateDomains(dm groups.Map) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.dm = dm
}
