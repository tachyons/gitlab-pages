package redirects

import (
	"context"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func generateRedirectsFile(dirPath string, count int) error {
	content := "/start.html /redirect.html 301\n"
	if feature.RedirectsPlaceholders.Enabled() {
		content += strings.Repeat("/foo/*/bar /foo/:splat/qux 200\n", count/2)
		content += strings.Repeat("/foo/:placeholder /qux/:placeholder 200\n", count/2)
	} else {
		content += strings.Repeat("/goto.html /target.html 301\n", count)
	}

	content += "/entrance.html /exit.html 301\n"

	return os.WriteFile(path.Join(dirPath, ConfigFile), []byte(content), 0600)
}

func benchmarkRedirectsRewrite(b *testing.B, redirectsCount int) {
	ctx := context.Background()

	root, tmpDir := testhelpers.TmpDir(b)

	err := generateRedirectsFile(tmpDir, redirectsCount)
	require.NoError(b, err)

	url, err := url.Parse("/entrance.html")
	require.NoError(b, err)

	redirects := ParseRedirects(ctx, root)
	require.NoError(b, redirects.error)

	for i := 0; i < b.N; i++ {
		_, _, err := redirects.Rewrite(url)
		require.NoError(b, err)
	}
}

func BenchmarkRedirectsRewrite_withoutPlaceholders(b *testing.B) {
	b.Run("10 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 10) })
	b.Run("100 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 100) })
	b.Run("1000 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 998) })
}

func BenchmarkRedirectsRewrite_PlaceholdersEnabled(b *testing.B) {
	b.Setenv(feature.RedirectsPlaceholders.EnvVariable, "true")

	b.Run("10 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 10) })
	b.Run("100 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 100) })
	b.Run("1000 redirects", func(b *testing.B) { benchmarkRedirectsRewrite(b, 998) })
}

func benchmarkRedirectsParseRedirects(b *testing.B, redirectsCount int) {
	ctx := context.Background()

	root, tmpDir := testhelpers.TmpDir(b)

	err := generateRedirectsFile(tmpDir, redirectsCount)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		redirects := ParseRedirects(ctx, root)
		require.NoError(b, redirects.error)
	}
}

func BenchmarkRedirectsParseRedirects(b *testing.B) {
	b.Run("10 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 10) })
	b.Run("100 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 100) })
	b.Run("1000 redirects", func(b *testing.B) { benchmarkRedirectsParseRedirects(b, 998) })
}
