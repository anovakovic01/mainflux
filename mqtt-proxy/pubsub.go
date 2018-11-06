package mqtt

import "github.com/mainflux/mainflux"

// PubSub enables message publish and subscribe.
type PubSub interface {
	mainflux.MessagePublisher

	// Subscribes to specified channel and receives messages over go channel.
	Subscribe(string, chan mainflux.RawMessage) (Unsubscribe, error)
}
