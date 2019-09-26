package serving

import "gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"

type diskServing struct {
	disk *disk.Serving
}

func (d *diskServing) ServeFileHTTP(h Handler) bool {
	return d.disk.ServeFileHTTP(h)
}

func (d *diskServing) ServeNotFoundHTTP(h Handler) {
	d.disk.ServeNotFoundHTTP(h)
}

func (d *diskServing) HasAcmeChallenge(h Handler, token string) bool {
	return d.disk.HasAcmeChallenge(h, token)
}
