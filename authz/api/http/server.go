package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	endpoint "github.com/go-kit/kit/endpoint"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
	"github.com/mainflux/mainflux/authz/api"
	"github.com/mainflux/mainflux/errors"
	log "github.com/mainflux/mainflux/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	authorization = "Authorization"
	contentType   = "application/json; charset=utf-8"
)

var logger log.Logger

// MakeHandler creates and returns new HTTP server handler.
func MakeHandler(svc api.Service, l log.Logger) http.Handler {
	logger = l

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/connect", kithttp.NewServer(
		svc.ConnectEndpoint,
		decodeConnectRequest,
		encodeResponse,
		opts...,
	))
	r.Post("/disconnect", kithttp.NewServer(
		svc.DisconnectEndpoint,
		decodeDisconnectRequest,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("authz"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeConnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := api.CreateConnectionsReq{
		Token: r.Header.Get("Authorization"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(fmt.Sprintf("Failed to decode create connection request: %s", err))
		return nil, errors.Wrap(authz.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := api.RemoveConnectionsReq{
		Token: r.Header.Get("Authorization"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(fmt.Sprintf("Failed to decode remove connection request: %s", err))
		return nil, errors.Wrap(authz.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		encodeError(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", contentType)

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err := err.(type) {
	case errors.Error:
		w.Header().Set("Content-Type", contentType)
		switch {
		case errors.Contains(err, authz.ErrUnauthorizedAccess):
			w.WriteHeader(http.StatusUnauthorized)
		case errors.Contains(err, authz.ErrAuthenticationFailed):
			w.WriteHeader(http.StatusUnauthorized)
		case errors.Contains(err, authz.ErrNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Contains(err, authz.ErrAlreadyExists):
			w.WriteHeader(http.StatusConflict)
		case errors.Contains(err, authz.ErrMalformedEntity):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(err, io.ErrUnexpectedEOF):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(err, io.EOF):
			w.WriteHeader(http.StatusBadRequest)
		}
		if err.Msg() != "" {
			json.NewEncoder(w).Encode(errorWrapper{Err: err.Msg()})
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type errorWrapper struct {
	Err string `json:"error"`
}
