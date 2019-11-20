// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/mainflux/mainflux/things"

var _ apiReq = (*identifyReq)(nil)

type apiReq interface {
	validate() error
}

type identifyReq struct {
	Token string `json:"token"`
}

func (req identifyReq) validate() error {
	if req.Token == "" {
		return things.ErrUnauthorizedAccess
	}

	return nil
}
