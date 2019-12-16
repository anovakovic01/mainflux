package mocks

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux/authz"
)

type authzMock struct {
	policies map[string]authz.Policy
}

// NewAuthZService returns mock authz service instance.
func NewAuthZService() authz.Service {
	return &authzMock{
		policies: map[string]authz.Policy{},
	}
}

func (am *authzMock) Authorize(context.Context, authz.Policy) error {
	panic("not implemented")
}

func (am *authzMock) Connect(_ context.Context, _ string, policies map[string]authz.Policy) (map[string]error, error) {
	for _, p := range policies {
		if _, ok := am.policies[key(p)]; ok {
			return nil, authz.ErrAlreadyExists
		}
		am.policies[key(p)] = p
	}

	return nil, nil
}

func (am *authzMock) Disconnect(_ context.Context, _ string, policies map[string]authz.Policy) (map[string]error, error) {
	for _, p := range policies {
		if _, ok := am.policies[key(p)]; !ok {
			return nil, authz.ErrNotFound
		}

		delete(am.policies, key(p))
	}

	return nil, nil
}

func (am *authzMock) AddChannels(context.Context, string, ...string) error {
	panic("not implemented")
}

func (am *authzMock) AddThings(context.Context, string, ...string) error {
	panic("not implemented")
}

func (am *authzMock) RemoveThing(context.Context, string, string) error {
	panic("not implemented")
}

func (am *authzMock) RemoveChannel(context.Context, string, string) error {
	panic("not implemented")
}

func key(p authz.Policy) string {
	return fmt.Sprintf("%s:%s:%s", p.Subject, p.Object, p.Action)
}
