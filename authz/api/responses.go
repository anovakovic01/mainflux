package api

// ErrorRes contains error response.
type ErrorRes struct {
	Err error
}

// Failed returns business logic error if there is any.
func (res ErrorRes) Failed() error {
	return res.Err
}

// BatchErrorRes contains error response for the batch operations that can
// partially fail.
type BatchErrorRes struct {
	Err  error            `json:"-"`
	Errs map[string]error `json:"errors"`
}

// Failed returns single business logic error if there is any. This method
// retuns a value only when there is common error for all of the batch
// operations.
func (res BatchErrorRes) Failed() error {
	return res.Err
}
