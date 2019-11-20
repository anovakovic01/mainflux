// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
)

var _ mainflux.MessagePublisher = (*adapterService)(nil)

type adapterService struct {
	pub    mainflux.MessagePublisher
	authz  authz.Service
	things mainflux.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(pub mainflux.MessagePublisher, authz authz.Service, things mainflux.ThingsServiceClient) mainflux.MessagePublisher {
	return &adapterService{
		pub:    pub,
		authz:  authz,
		things: things,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg mainflux.Message) error {
	res, err := as.things.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}
	thid := res.GetValue()

	p := authz.Policy{
		Subject: thid,
		Object:  msg.GetChannel(),
		Action:  "read",
	}
	if err := as.authz.Authorize(ctx, p); err != nil {
		return err
	}

	msg.Publisher = thid
	return as.pub.Publish(ctx, token, msg)
}
