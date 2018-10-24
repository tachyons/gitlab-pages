package admin

import (
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	context "golang.org/x/net/context"
)

func authFunc(token string) func(context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		return ctx, gitalyauth.CheckToken(ctx, token)
	}
}
