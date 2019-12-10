package authz

import (
	"context"
	"github.com/mainflux/mainflux/errors"
)

const (
	// ReadAct represents read action value.
	ReadAct = "read"

	// WriteAct represents write action value.
	WriteAct = "write"

	// OwningAct represents owning action value.
	OwningAct = "owning"
)

type service struct {
	enforcer Enforcer
	idp      IdentityProvider
}

// New creates and returns new authorization service instance.
func New(enforcer Enforcer, idp IdentityProvider) Service {
	return service{
		enforcer: enforcer,
		idp:      idp,
	}
}

func (svc service) Authorize(ctx context.Context, p Policy) error {
	r, err := svc.enforcer.Enforce(p.Subject, p.Object, p.Action)
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	if !r {
		return ErrUnauthorizedAccess
	}

	return nil
}

func (svc service) Connect(ctx context.Context, token string, ps map[string]Policy) (map[string]error, error) {
	userID, err := svc.idp.Identify(ctx, token)
	if err != nil {
		return nil, err
	}

	errs := map[string]error{}
	for k, p := range ps {
		if err := svc.isOwner(ctx, userID, p.Subject); err != nil {
			errs[k] = ErrUnauthorizedAccess
			continue
		}

		if err := svc.isOwner(ctx, userID, p.Object); err != nil {
			errs[k] = ErrUnauthorizedAccess
			continue
		}

		created, err := svc.enforcer.AddPolicy(p.Subject, p.Object, p.Action)
		if err != nil || !created {
			errs[k] = ErrFailedCreation
		}
	}

	return errs, nil
}

func (svc service) Disconnect(ctx context.Context, token string, ps map[string]Policy) (map[string]error, error) {
	userID, err := svc.idp.Identify(ctx, token)
	if err != nil {
		return nil, err
	}

	errs := map[string]error{}
	for k, p := range ps {
		if err := svc.isOwner(ctx, userID, p.Subject); err != nil {
			errs[k] = ErrUnauthorizedAccess
			continue
		}

		if err := svc.isOwner(ctx, userID, p.Object); err != nil {
			errs[k] = ErrUnauthorizedAccess
			continue
		}

		r, err := svc.enforcer.RemovePolicy(p.Subject, p.Object, p.Action)
		if err != nil || !r {
			errs[k] = ErrFailedRemoval
		}
	}

	return errs, nil
}

func (svc service) AddThings(ctx context.Context, owner string, ids ...string) error {
	return svc.addResource(ctx, owner, ids...)
}

func (svc service) AddChannels(ctx context.Context, owner string, ids ...string) error {
	return svc.addResource(ctx, owner, ids...)
}

func (svc service) RemoveChannel(ctx context.Context, owner, id string) error {
	if err := svc.isOwner(ctx, owner, id); err != nil {
		return err
	}

	r, err := svc.enforcer.RemoveFilteredPolicy(1, id)
	if err != nil {
		return errors.Wrap(ErrFailedRemoval, err)
	}
	if !r {
		return ErrFailedRemoval
	}

	r, err = svc.enforcer.RemoveFilteredPolicy(0, owner, id)
	if err != nil {
		return errors.Wrap(ErrFailedRemoval, err)
	}
	if !r {
		return ErrFailedRemoval
	}

	return nil
}

func (svc service) RemoveThing(ctx context.Context, owner, id string) error {
	if err := svc.isOwner(ctx, owner, id); err != nil {
		return err
	}

	r, err := svc.enforcer.RemoveFilteredPolicy(0, id)
	if err != nil {
		return errors.Wrap(ErrFailedRemoval, err)
	}
	if !r {
		return ErrNotFound
	}

	r, err = svc.enforcer.RemoveFilteredPolicy(0, owner, id)
	if err != nil {
		return errors.Wrap(ErrFailedRemoval, err)
	}
	if !r {
		return ErrNotFound
	}

	return nil
}

func (svc service) isOwner(ctx context.Context, owner string, ids ...string) error {
	for _, id := range ids {
		r, err := svc.enforcer.Enforce(owner, id, OwningAct)
		if err != nil {
			return errors.Wrap(ErrUnauthorizedAccess, err)
		}
		if !r {
			return ErrUnauthorizedAccess
		}
	}

	return nil
}

func (svc service) addResource(ctx context.Context, owner string, ids ...string) error {
	for _, id := range ids {
		r, err := svc.enforcer.AddPolicy(owner, id, OwningAct)
		if err != nil {
			return errors.Wrap(ErrFailedCreation, err)
		}
		if !r {
			return ErrAlreadyExists
		}
	}

	return nil
}
