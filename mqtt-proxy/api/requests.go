package api

import (
	"github.com/mainflux/mainflux"
	mqtt "github.com/mainflux/mainflux/mqtt-proxy"
)

type connectReq struct {
	password string
}

func (cr connectReq) Validate() error {
	if cr.password == "" {
		return errBadUsernameOrPassword
	}

	return nil
}

type publishReq struct {
	password string
	msg      mainflux.RawMessage
}

func (pr publishReq) Validate() error {
	if pr.password == "" {
		return errBadUsernameOrPassword
	}

	return nil
}

type subscribeReq struct {
	chanID string
}

func (sr subscribeReq) Validate() error {
	if sr.chanID == "" {
		return mqtt.ErrMalformedData
	}

	return nil
}
