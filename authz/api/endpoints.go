package api

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/authz"
)

// ErrInvalidReq indicates that client has sent an invalid request.
var ErrInvalidReq = errors.New("received invalid request")

// MakeAuthorizeEndpoint contains generic authorization endpoint flow.
func MakeAuthorizeEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AuthZReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		p := authz.Policy{
			Subject: req.Sub,
			Object:  req.Obj,
			Action:  req.Act,
		}

		err := svc.Authorize(ctx, p)
		return ErrorRes{Err: err}, nil
	}
}

// MakeConnectEndpoint contains generic add policy endpoint flow.
func MakeConnectEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateConnectionsReq)
		res := BatchErrorRes{
			Errs: map[string]error{},
		}

		ps := map[string]authz.Policy{}
		for k, v := range req.Connections {
			if err := v.validate(); err != nil {
				res.Errs[k] = err
				continue
			}

			ps[k] = authz.Policy{
				Subject: v.Sub,
				Object:  v.Obj,
				Action:  v.Act,
			}
		}

		errs, err := svc.Connect(ctx, req.Token, ps)
		if err != nil {
			res.Err = err
			return res, nil
		}

		for k, e := range errs {
			res.Errs[k] = e
		}

		return res, nil
	}
}

// MakeDisconnectEndpoint contains generic remove policy endpoint flow.
func MakeDisconnectEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RemoveConnectionsReq)
		res := BatchErrorRes{
			Errs: map[string]error{},
		}

		ps := map[string]authz.Policy{}
		for k, v := range req.Connections {
			if err := v.validate(); err != nil {
				res.Errs[k] = err
				continue
			}

			ps[k] = authz.Policy{
				Subject: v.Sub,
				Object:  v.Obj,
				Action:  v.Act,
			}
		}

		errs, err := svc.Disconnect(ctx, req.Token, ps)
		if err != nil {
			res.Err = err
			return res, nil
		}

		for k, e := range errs {
			res.Errs[k] = e
		}

		return res, nil
	}
}

// MakeAddThingsEndpoint contains generic add things endpoint flow.
func MakeAddThingsEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddThingsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.AddThings(ctx, req.Owner, req.IDs...)
		return ErrorRes{Err: err}, nil
	}
}

// MakeAddChannelsEndpoint contains generic add channels endpoint flow.
func MakeAddChannelsEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddChannelsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.AddChannels(ctx, req.Owner, req.IDs...)
		return ErrorRes{Err: err}, nil
	}
}

// MakeRemoveChannelEndpoint contains generic remove channel endpoint flow.
func MakeRemoveChannelEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RemoveChannelReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.RemoveChannel(ctx, req.Owner, req.ID)
		return ErrorRes{Err: err}, nil
	}
}

// MakeRemoveThingEndpoint contains generic remove thing endpoint flow.
func MakeRemoveThingEndpoint(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RemoveThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.RemoveThing(ctx, req.Owner, req.ID)
		return ErrorRes{Err: err}, nil
	}
}
