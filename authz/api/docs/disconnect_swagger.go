package docs

import "github.com/mainflux/mainflux/authz/api"

// swagger:route POST /disconnect connection removeConnectionsEndpoint
// Remove a write or read access to the given subjects over given objects.
// responses:
//   200: batchErrorResponse
//   400: description: Received invalid request.
//   401: description: Authorization failed.
//   409: description: Connection already exists.
//   500: description: Unexpected error happened.

// swagger:parameters removeConnectionsEndpoint
type removeConnectionsReqWrapper struct {
	// in:body
	Body api.RemoveConnectionsReq
}
