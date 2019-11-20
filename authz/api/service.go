package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	"github.com/mainflux/mainflux/authz"
	opentracing "github.com/opentracing/opentracing-go"
)

var _ authz.Service = (*Service)(nil)

// Service contains all of the endpoints that this service can serve. This
// service is implementing authz service in order to be used as interface for
// the client library.
type Service struct {
	AuthorizeEndpoint     endpoint.Endpoint
	ConnectEndpoint       endpoint.Endpoint
	DisconnectEndpoint    endpoint.Endpoint
	AddThingsEndpoint     endpoint.Endpoint
	AddChannelsEndpoint   endpoint.Endpoint
	RemoveChannelEndpoint endpoint.Endpoint
	RemoveThingEndpoint   endpoint.Endpoint
}

// New returns new generic service instance.
func New(svc authz.Service, tracer opentracing.Tracer) Service {
	authorize := MakeAuthorizeEndpoint(svc)
	authorize = kitot.TraceServer(tracer, "authorize")(authorize)

	connect := MakeConnectEndpoint(svc)
	connect = kitot.TraceServer(tracer, "connect")(connect)

	disconnect := MakeDisconnectEndpoint(svc)
	disconnect = kitot.TraceServer(tracer, "disconnect")(disconnect)

	addThings := MakeAddThingsEndpoint(svc)
	addThings = kitot.TraceServer(tracer, "add_things")(addThings)

	addChannels := MakeAddChannelsEndpoint(svc)
	addChannels = kitot.TraceServer(tracer, "add_channels")(addChannels)

	removeChannel := MakeRemoveChannelEndpoint(svc)
	removeChannel = kitot.TraceServer(tracer, "remove_channel")(removeChannel)

	removeThing := MakeRemoveThingEndpoint(svc)
	removeThing = kitot.TraceServer(tracer, "remove_thing")(removeThing)

	return Service{
		AuthorizeEndpoint:     authorize,
		ConnectEndpoint:       connect,
		DisconnectEndpoint:    disconnect,
		AddThingsEndpoint:     addThings,
		AddChannelsEndpoint:   addChannels,
		RemoveChannelEndpoint: removeChannel,
		RemoveThingEndpoint:   removeThing,
	}
}

// Authorize checks if given entity has required access rights to the given
// object.
func (svc Service) Authorize(ctx context.Context, p authz.Policy) error {
	req := AuthZReq{
		Sub: p.Subject,
		Obj: p.Object,
		Act: p.Action,
	}

	resp, err := svc.AuthorizeEndpoint(ctx, req)
	if err != nil {
		return err
	}

	res := resp.(ErrorRes)
	return res.Err
}

// Connect connects subject (thing) with the given object (channel).
func (svc Service) Connect(ctx context.Context, token string, ps map[string]authz.Policy) (map[string]error, error) {
	req := CreateConnectionsReq{
		Token: token,
	}
	for k, v := range ps {
		req.Connections[k] = ConnectReq{
			Sub: v.Subject,
			Obj: v.Object,
			Act: v.Action,
		}
	}

	resp, err := svc.ConnectEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}

	res := resp.(BatchErrorRes)
	return res.Errs, res.Err
}

// Disconnect disconnects subject (thing) from the given object (channel).
func (svc Service) Disconnect(ctx context.Context, token string, ps map[string]authz.Policy) (map[string]error, error) {
	req := RemoveConnectionsReq{
		Token: token,
	}
	for k, v := range ps {
		req.Connections[k] = ConnectReq{
			Sub: v.Subject,
			Obj: v.Object,
			Act: v.Action,
		}
	}

	resp, err := svc.DisconnectEndpoint(ctx, req)
	if err != nil {
		return nil, err
	}

	res := resp.(BatchErrorRes)
	return res.Errs, res.Err
}

// AddThings adds multiple things at once along with their owner.
func (svc Service) AddThings(ctx context.Context, owner string, ids ...string) error {
	req := AddThingsReq{
		Owner: owner,
		IDs:   ids,
	}

	resp, err := svc.AddThingsEndpoint(ctx, req)
	if err != nil {
		return err
	}

	res := resp.(ErrorRes)
	return res.Err
}

// AddChannels adds multiple channels at once along with their owner.
func (svc Service) AddChannels(ctx context.Context, owner string, ids ...string) error {
	req := AddChannelsReq{
		Owner: owner,
		IDs:   ids,
	}

	resp, err := svc.AddChannelsEndpoint(ctx, req)
	if err != nil {
		return err
	}

	res := resp.(ErrorRes)
	return res.Err
}

// RemoveChannel removes all of the object (channel) connections.
func (svc Service) RemoveChannel(ctx context.Context, owner, id string) error {
	req := RemoveChannelReq{
		ID: id,
	}

	resp, err := svc.RemoveChannelEndpoint(ctx, req)
	if err != nil {
		return err
	}

	res := resp.(ErrorRes)
	return res.Err
}

// RemoveThing removes all of the subject (thing) connections.
func (svc Service) RemoveThing(ctx context.Context, owner, id string) error {
	req := RemoveThingReq{
		ID: id,
	}

	resp, err := svc.RemoveThingEndpoint(ctx, req)
	if err != nil {
		return err
	}

	res := resp.(ErrorRes)
	return res.Err
}
