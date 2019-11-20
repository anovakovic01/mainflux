package authz

import "context"

// IdentityProvider identifies an entity based on it's token.
type IdentityProvider interface {
	Identify(context.Context, string) (string, error)
}
