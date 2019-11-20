// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things"
	context "golang.org/x/net/context"
)

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		id, err := svc.Identify(ctx, req.key)
		if err != nil {
			return identityRes{err: err}, err
		}
		return identityRes{id: id, err: nil}, nil
	}
}
