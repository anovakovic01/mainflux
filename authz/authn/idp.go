package authn

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
	"github.com/mainflux/mainflux/errors"
)

type authnIDP struct {
	client mainflux.AuthNServiceClient
}

// New returns new identity provider instance that can identify users based on
// the given token.
func New(client mainflux.AuthNServiceClient) authz.IdentityProvider {
	return authnIDP{
		client: client,
	}
}

func (idp authnIDP) Identify(ctx context.Context, token string) (string, error) {
	req := &mainflux.Token{Value: token}
	id, err := idp.client.Identify(ctx, req)
	if err != nil {
		return "", errors.Wrap(authz.ErrUnauthorizedAccess, err)
	}

	return id.GetValue(), nil
}
