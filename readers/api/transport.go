// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
	"github.com/mainflux/mainflux/readers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType = "application/json"
	defLimit    = 10
	defOffset   = 0
)

var (
	errInvalidRequest     = errors.New("received invalid request")
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	auth                  mainflux.ThingsServiceClient
	queryFields           = []string{"subtopic", "publisher", "protocol", "name", "value", "v", "vs", "vb", "vd"}
)

type server struct {
	authz authz.Service
	authn mainflux.ThingsServiceClient
}

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, authz authz.Service, authn mainflux.ThingsServiceClient, svcName string) http.Handler {
	s := server{
		authz: authz,
		authn: authn,
	}

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()
	mux.Get("/channels/:chanID/messages", kithttp.NewServer(
		listMessagesEndpoint(s, svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/version", mainflux.Version(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	chanID := bone.GetValue(r, "chanID")
	if chanID == "" {
		return nil, errInvalidRequest
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, errUnauthorizedAccess
	}

	offset, err := getQuery(r, "offset", defOffset)
	if err != nil {
		return nil, err
	}

	limit, err := getQuery(r, "limit", defLimit)
	if err != nil {
		return nil, err
	}

	query := map[string]string{}
	for _, name := range queryFields {
		if value := bone.GetQuery(r, name); len(value) == 1 {
			query[name] = value[0]
		}
	}

	req := listMessagesReq{
		token:  token,
		chanID: chanID,
		offset: offset,
		limit:  limit,
		query:  query,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err {
	case nil:
	case errInvalidRequest:
		w.WriteHeader(http.StatusBadRequest)
	case errUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s server) authorize(token string, chanID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := s.authn.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		e, ok := status.FromError(err)
		if ok && e.Code() == codes.PermissionDenied {
			return errUnauthorizedAccess
		}
		return err
	}

	p := authz.Policy{
		Subject: id.GetValue(),
		Object:  chanID,
		Action:  "read",
	}
	return s.authz.Authorize(ctx, p)
}

func getQuery(req *http.Request, name string, fallback uint64) (uint64, error) {
	vals := bone.GetQuery(req, name)
	if len(vals) == 0 {
		return fallback, nil
	}

	if len(vals) > 1 {
		return 0, errInvalidRequest
	}

	val, err := strconv.ParseUint(vals[0], 10, 64)
	if err != nil {
		return 0, errInvalidRequest
	}

	return uint64(val), nil
}
