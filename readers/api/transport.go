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
	"github.com/mainflux/mainflux/readers"
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
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, tc mainflux.ThingsServiceClient) http.Handler {
	auth = tc

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()
	mux.Get("/channels/:chanID/messages", kithttp.NewServer(
		listMessagesEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	chanID, err := strconv.ParseUint(bone.GetValue(r, "chanID"), 10, 64)
	if err != nil {
		return nil, errInvalidRequest
	}

	if err := authorize(r, chanID); err != nil {
		return nil, err
	}

	offset := getQuery(r, "offset", defOffset)
	limit := getQuery(r, "limit", defLimit)

	req := listMessagesReq{
		chanID: chanID,
		offset: offset,
		limit:  limit,
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
	case readers.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func authorize(r *http.Request, chanID uint64) error {
	token := r.Header.Get("Authorization")
	if token == "" {
		return errUnauthorizedAccess
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := auth.CanAccess(ctx, &mainflux.AccessReq{Token: token, ChanID: chanID})
	if err != nil {
		e, ok := status.FromError(err)
		if ok && e.Code() == codes.PermissionDenied {
			return errUnauthorizedAccess
		}
		return err
	}

	return nil
}

func getQuery(req *http.Request, name string, fallback uint64) uint64 {
	vals := bone.GetQuery(req, name)
	if len(vals) == 0 {
		return fallback
	}

	val, err := strconv.ParseUint(vals[0], 10, 64)
	if err != nil {
		return fallback
	}

	return uint64(val)
}
