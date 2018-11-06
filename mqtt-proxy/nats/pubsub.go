package nats

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	mqtt "github.com/mainflux/mainflux/mqtt-proxy"
	broker "github.com/nats-io/go-nats"
)

const (
	prefix   = "channel"
	protocol = "mqtt"
)

var _ mqtt.PubSub = (*pubsub)(nil)

type pubsub struct {
	nc *broker.Conn
}

func (ps pubsub) Publish(msg mainflux.RawMessage) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	return ps.nc.Publish(fmt.Sprintf("%s.%d", prefix, msg.Channel), data)
}

func (ps pubsub) Subscribe(chanID string, msgCh chan mainflux.RawMessage) (mqtt.Unsubscribe, error) {
	sub, err := ps.nc.Subscribe(fmt.Sprintf("%s.%s", prefix, chanID), func(msg *broker.Msg) {
		if msg == nil {
			return
		}

		var rawMsg mainflux.RawMessage
		if err := proto.Unmarshal(msg.Data, &rawMsg); err != nil {
			return
		}

		if rawMsg.GetProtocol() == protocol {
			return
		}

		msgCh <- rawMsg
	})

	return func() error {
		close(msgCh)
		return sub.Unsubscribe()
	}, err
}
