package disk

import (
	"strings"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// Domains struct represents a map of all domains supported by pages. It is
// currently reading them from disk.
type Disk struct {
	dm   Map
	lock *sync.RWMutex
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func New() *Disk {
	return &Disk{
		lock: &sync.RWMutex{},
	}
}

// GetDomain returns a domain from the domains map
func (d *Disk) GetDomain(host string) *domain.Domain {
	host = strings.ToLower(host)
	d.lock.RLock()
	defer d.lock.RUnlock()
	domain, _ := d.dm[host]

	return domain
}

// HasDomain checks for presence of a domain in the domains map
func (d *Disk) HasDomain(host string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()

	host = strings.ToLower(host)
	_, isPresent := d.dm[host]

	return isPresent
}

// Ready checks if the domains source is ready for work
func (d *Disk) Ready() bool {
	return d.dm != nil
}

// Watch starts the domain source, in this case it is reading domains from
// groups on disk concurrently
func (d *Disk) Watch(rootDomain string) {
	go watch(rootDomain, d.updateDomains, time.Second)
}

func (d *Disk) updateDomains(dm Map) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.dm = dm
}
