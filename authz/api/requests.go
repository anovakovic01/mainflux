package api

import "github.com/mainflux/mainflux/authz"

// AuthZReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write)
type AuthZReq struct {
	Sub string
	Obj string
	Act string
}

func (req AuthZReq) validate() error {
	if req.Sub == "" {
		return ErrInvalidReq
	}

	if req.Obj == "" {
		return ErrInvalidReq
	}

	if req.Act == "" {
		return ErrInvalidReq
	}

	return nil
}

// CreateConnectionsReq contains data necessary for connecting things and
// channels.
type CreateConnectionsReq struct {
	Token       string                `json:"-"`
	Connections map[string]ConnectReq `json:"connections"`
}

// ConnectReq represents add policy request. It contains all of the fields
// that are needed in order to create a policy.
type ConnectReq struct {
	Sub string `json:"sub"`
	Obj string `json:"obj"`
	Act string `json:"act"`
}

func (req ConnectReq) validate() error {
	if req.Sub == "" {
		return ErrInvalidReq
	}

	if req.Obj == "" {
		return ErrInvalidReq
	}

	if req.Act != authz.ReadAct && req.Act != authz.WriteAct {
		return ErrInvalidReq
	}

	return nil
}

// RemoveConnectionsReq contains data necessary for disconnecting things and
// channels.
type RemoveConnectionsReq struct {
	Token       string                `json:"-"`
	Connections map[string]ConnectReq `json:"connections"`
}

// AddThingsReq represents add multiple things request. If contains just
// things IDs and owner ID.
type AddThingsReq struct {
	Owner string
	IDs   []string
}

func (req AddThingsReq) validate() error {
	if req.Owner == "" {
		return ErrInvalidReq
	}

	if len(req.IDs) == 0 {
		return ErrInvalidReq
	}

	return nil
}

// AddChannelsReq represents add multiple channels request. If contains just
// channels IDs and owner ID.
type AddChannelsReq struct {
	Owner string
	IDs   []string
}

func (req AddChannelsReq) validate() error {
	if req.Owner == "" {
		return ErrInvalidReq
	}

	if len(req.IDs) == 0 {
		return ErrInvalidReq
	}

	return nil
}

// RemoveChannelReq represents remove channel request. If contains channel ID
// by which channel connections will be removed.
type RemoveChannelReq struct {
	Owner string
	ID    string
}

func (req RemoveChannelReq) validate() error {
	if req.Owner == "" || req.ID == "" {
		return ErrInvalidReq
	}

	return nil
}

// RemoveThingReq represents remove thing request. If contains thing ID
// by which thing connections will be removed.
type RemoveThingReq struct {
	Owner string
	ID    string
}

func (req RemoveThingReq) validate() error {
	if req.Owner == "" || req.ID == "" {
		return ErrInvalidReq
	}

	return nil
}
