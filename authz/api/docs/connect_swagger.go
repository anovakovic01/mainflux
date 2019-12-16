package docs

import "github.com/mainflux/mainflux/authz/api"

// swagger:route POST /connect connection createConnectionsEndpoint
// Gives a write or read access to the given subjects over given objects.
// responses:
//   200: batchErrorResponse
//   400: description: Received invalid request.
//   401: description: Authorization failed.
//   409: description: Connection already exists.
//   500: description: Unexpected error happened.

// swagger:parameters createConnectionsEndpoint
type createConnectionsReqWrapper struct {
	// This request contains set of rules. Every rule contains subject, object
	// and action.
	// in:body
	Body api.CreateConnectionsReq
}

// Response body will contain a map of errors. Keys for this map will be
// used as correlation IDs in order to get errors for specific rules. If
// operation fails in general, then you will get error description under
// "error" key.
// swagger:response batchErrorResponse
type batchErrorResWrapper struct {
	// in:body
	Body api.BatchErrorRes
}
