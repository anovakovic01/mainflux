package http

import (
	"context"
	"encoding/json"
	"net/http"

	endpoint "github.com/go-kit/kit/endpoint"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	authorization = "Authorization"
	contentType   = "application/json; charset=utf-8"
)

// MakeHandler creates and returns new HTTP server handler.
func MakeHandler(svc api.Service) http.Handler {
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
		return nil, err
	}

	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := api.RemoveConnectionsReq{
		Token: r.Header.Get("Authorization"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
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
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Err: err.Error()})
}

func err2code(err error) int {
	return http.StatusInternalServerError
}

type errorWrapper struct {
	Err string `json:"error"`
}
