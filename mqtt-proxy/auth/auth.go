//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package auth has identity provider implementation.
package auth

import (
	"context"
	"strconv"
	"time"

	"github.com/mainflux/mainflux"
	mqtt "github.com/mainflux/mainflux/mqtt-proxy"
)

var _ mqtt.IdentityProvider = (*identityProvider)(nil)

type identityProvider struct {
	client mainflux.ThingsServiceClient
}

// New returns new identity provider instance.
func New(client mainflux.ThingsServiceClient) mqtt.IdentityProvider {
	return identityProvider{client: client}
}

func (idp identityProvider) CanAccess(chanID, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cid, err := strconv.ParseUint(chanID, 10, 64)
	if err != nil {
		return "", mqtt.ErrMalformedData
	}

	req := &mainflux.AccessReq{
		Token:  key,
		ChanID: cid,
	}
	tid, err := idp.client.CanAccess(ctx, req)
	if err != nil {
		return "", mqtt.ErrUnauthorized
	}

	return strconv.FormatUint(tid.GetValue(), 10), nil
}

func (idp identityProvider) Identify(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &mainflux.Token{
		Value: key,
	}
	tid, err := idp.client.Identify(ctx, req)
	if err != nil {
		return "", mqtt.ErrUnauthorized
	}

	return strconv.FormatUint(tid.GetValue(), 10), nil
}
