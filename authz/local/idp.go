package local

import (
	"context"

	"github.com/mainflux/mainflux/authz"
)

type idp struct {
	id    string
	token string
}

// NewIDP returns new local instance in single user mode.
func NewIDP(id, token string) authz.IdentityProvider {
	return idp{
		id:    id,
		token: token,
	}
}

func (idp idp) Identify(_ context.Context, token string) (string, error) {
	if token != idp.token {
		return "", authz.ErrAuthenticationFailed
	}

	return idp.id, nil
}
