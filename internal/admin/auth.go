package admin

import (
	context "golang.org/x/net/context"

	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
)

func authFunc(token string) func(context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		return ctx, gitalyauth.CheckToken(ctx, token)
	}
}
