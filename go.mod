module gitlab.com/gitlab-org/gitlab-pages

// before bumping this:
// - update the minimum version used in ci
// - make sure the internal/vfs/serving package is synced
//   with upstream
go 1.16

require (
	github.com/golang-jwt/jwt/v4 v4.1.0
	github.com/golang/mock v1.3.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/karlseguin/ccache/v2 v2.0.6
	github.com/namsral/flag v1.7.4-pre
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pires/go-proxyproto v0.2.0
	github.com/prometheus/client_golang v1.6.0
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tj/assert v0.0.3 // indirect
	github.com/tj/go-redirects v0.0.0-20180508180010-5c02ead0bbc5
	gitlab.com/gitlab-org/go-mimedb v1.45.0
	gitlab.com/gitlab-org/labkit v1.3.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
)
