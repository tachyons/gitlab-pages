package gitlabstub

import (
	"net/http"
	"time"
)

type config struct {
	pagesHandler http.HandlerFunc
	pagesRoot    string
	delay        time.Duration
}

type Option func(*config)

func WithPagesHandler(ph http.HandlerFunc) Option {
	return func(sc *config) {
		sc.pagesHandler = ph
	}
}

func WithPagesRoot(pagesRoot string) Option {
	return func(sc *config) {
		sc.pagesRoot = pagesRoot
	}
}

func WithDelay(delay time.Duration) Option {
	return func(sc *config) {
		sc.delay = delay
	}
}
