//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things"
)

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func canAccessEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(canAccessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccess(req.ChanID, req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}