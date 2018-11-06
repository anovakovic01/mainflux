package mqtt

import (
	"errors"
	"strconv"

	"github.com/mainflux/mainflux"
)

var (
	// ErrUnauthorized failed to authorize client with given credentials.
	ErrUnauthorized = errors.New("invalid credentials")

	// ErrMalformedData indicates that invalid data was received.
	ErrMalformedData = errors.New("received invalid data")

	// ErrNotFound indicates that requested entity doesn't exist.
	ErrNotFound = errors.New("entity not found")
)

var _ Service = (*service)(nil)

// Service represents clients session.
type Service interface {
	// Connect creates MQTT session and connects to MQTT broker.
	Connect(string) error

	// Subscribe to specified MQTT topic.
	Subscribe(string) (chan mainflux.RawMessage, error)

	// Unsubscribe from mainflux channel.
	Unsubscribe(string) error

	// Publish message to specified MQTT topic.
	Publish(msg mainflux.RawMessage) error
}

// Service contains MQTT data.
type service struct {
	idp      IdentityProvider
	password string
	ps       PubSub
	subs     map[string]Unsubscribe
}

// Unsubscribe represents method that unsubscribes from topic.
type Unsubscribe func() error

func (svc *service) Connect(password string) error {
	if _, err := svc.idp.Identify(password); err != nil {
		return err
	}

	svc.password = password

	return nil
}

func (svc service) Subscribe(chanID string) (chan mainflux.RawMessage, error) {
	_, err := svc.idp.CanAccess(chanID, svc.password)
	if err != nil {
		return nil, err
	}

	var msgCh chan mainflux.RawMessage
	unsub, err := svc.ps.Subscribe(chanID, msgCh)
	if err != nil {
		return nil, err
	}
	svc.subs[chanID] = unsub

	return msgCh, nil
}

func (svc service) Unsubscribe(chanID string) error {
	unsub, ok := svc.subs[chanID]
	if !ok {
		return ErrNotFound
	}

	return unsub()
}

func (svc service) Publish(msg mainflux.RawMessage) error {
	chanID := strconv.FormatUint(msg.Channel, 10)
	id, err := svc.idp.CanAccess(chanID, svc.password)
	if err != nil {
		return err
	}

	msg.Publisher, err = strconv.ParseUint(id, 10, 64)
	if err != nil {
		return ErrMalformedData
	}

	return svc.ps.Publish(msg)
}
