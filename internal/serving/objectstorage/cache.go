package objectstorage

import (
	"context"
	"errors"
	"sync"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/zip/reader"
)

var (
	errNotExists = errors.New("domain does not exist")
)

type inMemory struct {
	mu     *sync.Mutex
	reader *reader.Reader
}

func newInMemoryCache() *inMemory {
	return &inMemory{
		mu: new(sync.Mutex),
	}
}
func (i *inMemory) Set(ctx context.Context, cancel func(), reader *reader.Reader) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.reader = reader

	// clears the reader when the ctx is done or cancelled
	go i.clear(ctx, cancel)
}
func (i *inMemory) Reader() (*reader.Reader, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.reader == nil {
		return nil, errNotExists
	}

	return i.reader, nil
}

func (i *inMemory) clear(ctx context.Context, cancel func()) {
	<-ctx.Done()
	cancel()

	i.mu.Lock()
	defer i.mu.Unlock()

	logrus.Debug("removing expired reader")
	i.reader = nil
}
