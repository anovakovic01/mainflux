//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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

type canAccessReq struct {
	Token  string `json:"token"`
	ChanID string `json:"chan_id"`
}

func (req canAccessReq) validate() error {
	if req.Token == "" || req.ChanID == "" {
		return things.ErrUnauthorizedAccess
	}

	return nil
}
