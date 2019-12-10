package authz

import (
	"context"
	"github.com/mainflux/mainflux/errors"
)

var (
	// ErrUnauthorizedAccess indicates that given request hasn't passed any
	// policy.
	ErrUnauthorizedAccess = errors.New("unauthorized access")

	// ErrFailedCreation indicates that service failed to create the given
	// policy.
	ErrFailedCreation = errors.New("failed to create policy")

	// ErrFailedRemoval indicates that the service failed to remove the given
	// policy.
	ErrFailedRemoval = errors.New("failed to remove policy")

	// ErrAuthenticationFailed indicates that the given authentication token was
	// invalid.
	ErrAuthenticationFailed = errors.New("failed to authenticate given entity")

	// ErrAlreadyExists indcidates that the given entity already exists.
	ErrAlreadyExists = errors.New("Entity already exists")

	// ErrNotFound indicates that the required entity doesn't exist.
	ErrNotFound = errors.New("Required entity not found")

	// ErrMalformedEntity indicates that the received entity is malformed.
	ErrMalformedEntity = errors.New("Received invalid entity")
)

// Policy contains structured policy description.
type Policy struct {
	Subject string
	Object  string
	Action  string
}

// Service contains API that is used for access control purposes.
type Service interface {
	// Authorize checks if the given subject has the right to execute given
	// action over given object.
	Authorize(context.Context, Policy) error

	// AddPolicy creates new policy entity. If the creation fails, it will
	// return an error.
	Connect(context.Context, string, map[string]Policy) (map[string]error, error)

	// RemovePolicy removes an existing policy. If the removal fails, it will
	// return an error.
	Disconnect(context.Context, string, map[string]Policy) (map[string]error, error)

	// AddThings adds things under the same owner.
	AddThings(context.Context, string, ...string) error

	// AddChannels adds channels under the same owner.
	AddChannels(context.Context, string, ...string) error

	// RemoveThing removes all of the thing connections. If the removal fails,
	// it will return an error.
	RemoveThing(context.Context, string, string) error

	// RemoveChannel removes all of the channel connections. If the removal
	// fails, it will return an error.
	RemoveChannel(context.Context, string, string) error
}
