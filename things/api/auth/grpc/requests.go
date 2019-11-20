// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/mainflux/mainflux/things"

type identifyReq struct {
	key string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return things.ErrMalformedEntity
	}

	return nil
}
