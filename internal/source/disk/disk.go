package disk

import (
	"strings"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// Disk struct represents a map of all domains supported by pages that are
// stored on a disk with corresponding `config.json`.
type Disk struct {
	dm   Map
	lock *sync.RWMutex
}

// New is a factory method for the Disk source. It is initializing a mutex. It
// should not initialize `dm` as we later check the readiness by comparing it
// with a nil value.
func New(rootDomain string) *Disk {
	d := &Disk{
		lock: &sync.RWMutex{},
	}

	// start reading domain configuration from disk concurrently
	go Watch(rootDomain, d.updateDomains, time.Second)

	return d
}

// GetDomain returns a domain from the domains map if it exists
func (d *Disk) GetDomain(host string) (*domain.Domain, error) {
	host = strings.ToLower(host)

	d.lock.RLock()
	defer d.lock.RUnlock()

	return d.dm[host], nil
}

// IsReady checks if the domains source is ready for work. The disk source is
// ready after traversing entire filesystem and reading all domains'
// configuration files.
func (d *Disk) IsReady() bool {
	return d.dm != nil
}

func (d *Disk) updateDomains(dm Map) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.dm = dm
}
