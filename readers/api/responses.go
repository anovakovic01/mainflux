package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var _ mainflux.Response = (*listMessagesRes)(nil)

type listMessagesRes struct {
	messages []mainflux.Message
}

func (res listMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMessagesRes) Code() int {
	return http.StatusOK
}

func (res listMessagesRes) Empty() bool {
	return false
}
