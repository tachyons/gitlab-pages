package objectstorage

import (
	"context"
	"errors"
	"sync"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/zipartifacts/reader"
)

var (
	errNotExists = errors.New("domain does not exist")
)

type archive struct {
	reader *reader.Reader
}
type inMemory struct {
	mu *sync.Mutex
	// TODO reuse per domain
	archive *archive
}

func newInMemoryCache() *inMemory {
	return &inMemory{
		mu:      new(sync.Mutex),
		archive: &archive{},
	}
}
func (i *inMemory) Set(ctx context.Context, reader *reader.Reader) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.archive = &archive{
		reader: reader,
	}
	go i.clear(ctx)

	return
}
func (i *inMemory) Reader() (*reader.Reader, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.archive == nil || i.archive.reader == nil {
		return nil, errNotExists
	}

	return i.archive.reader, nil
}

func (i *inMemory) clear(ctx context.Context) {
	<-ctx.Done()

	i.mu.Lock()
	defer i.mu.Unlock()

	logrus.Debug("removing expired reader")
	i.archive.reader = nil
}
