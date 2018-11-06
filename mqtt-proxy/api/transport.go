//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	mqtt "github.com/mainflux/mainflux/mqtt-proxy"
)

const (
	prefix   = "channels."
	protocol = "mqtt"
)

var brokerURL string

var errBadUsernameOrPassword = errors.New("bad username or password")

// Session contains MQTT data.
type Session struct {
	id string
	wg *sync.WaitGroup
	s  mqtt.Service
}

// Listen for MQTT connection.
func Listen(ln net.Listener, factory func() mqtt.Service) {
	for {
		// Listen for an incoming connection.
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}

		go listen(conn, factory)
	}
}

func listen(conn net.Conn, factory func() mqtt.Service) {
	broker, err := net.Dial("tcp", brokerURL)
	if err != nil {
		log.Println("Dial failed: ", brokerURL, err)
		return
	}
	defer broker.Close()
	defer conn.Close()

	session := Session{
		wg: &sync.WaitGroup{},
		s:  factory(),
	}

	session.wg.Add(2)

	go session.forwardToBroker(conn, broker)
	go session.forwardToClient(broker, conn)

	session.wg.Wait()
}

func (session Session) forwardToBroker(r net.Conn, w net.Conn) {
	defer session.wg.Done()

	for {
		cp, err := packets.ReadPacket(r)
		if err != nil {
			log.Println("Reading MQTT packet failed: ", err)
			break
		}

		switch p := cp.(type) {
		case *packets.ConnectPacket:
			err = session.connEndpoint(p, r, w)
		case *packets.SubscribePacket:
			err = session.subEndpoint(p, r, w)
		case *packets.UnsubscribePacket:
			session.unsubEndpoint(p, r, w)
		case *packets.PublishPacket:
			err = session.pubEndpoint(p, r, w)
		default:
			err = nil
		}

		if err != nil {
			log.Println("Received invalid packet: ", err)
			break
		}

		if err := cp.Write(w); err != nil {
			log.Println("Failed to write MQTT packet: ", err)
			break
		}
	}
}

func (session Session) forwardToClient(r net.Conn, w net.Conn) {
	defer session.wg.Done()

	for {
		cp, err := packets.ReadPacket(r)
		if err != nil {
			log.Println("Reading MQTT packet failed: ", err)
			break
		}

		if err := cp.Write(w); err != nil {
			log.Println("Failed to write MQTT packet: ", err)
			break
		}
	}
}

// If connect fails, then should return error anyway (doesn't matter if write to
// client was successful or not).
func (session Session) connEndpoint(p *packets.ConnectPacket, r net.Conn, w net.Conn) error {
	req := decodeConn(p)
	if err := req.Validate(); err != nil {
		res := connectRes{err: err}
		encodeConn(res, r)
		return err
	}

	if err := session.s.Connect(req.password); err != nil {
		res := connectRes{err: err}
		encodeConn(res, r)
		return err
	}

	return nil
}

// If writing to client was unsuccessful, should return error.
func (session Session) subEndpoint(p *packets.SubscribePacket, r net.Conn, w net.Conn) error {
	topicsNum := len(p.Topics)
	req, err := decodeSub(p)
	if err != nil {
		res := subscribeRes{
			id:        p.MessageID,
			topicsNum: topicsNum,
			err:       err,
		}
		return encodeSub(res, r)
	}

	if err := req.Validate(); err != nil {
		res := subscribeRes{
			id:        p.MessageID,
			topicsNum: topicsNum,
			err:       err,
		}
		return encodeSub(res, r)
	}

	var msgCh chan mainflux.RawMessage
	msgCh, err = session.s.Subscribe(req.chanID)
	if err != nil {
		res := subscribeRes{
			id:        p.MessageID,
			topicsNum: topicsNum,
			err:       err,
		}
		return encodeSub(res, r)
	}

	go receiveMessages(msgCh, p, r)
	return nil
}

func (session Session) unsubEndpoint(p *packets.UnsubscribePacket, r net.Conn, w net.Conn) {
	if len(p.Topics) != 1 {
		// TODO: handle error
		return
	}
	session.s.Unsubscribe(p.Topics[0])
}

// Return every error (if anything goes wrong, should break connection).
func (session Session) pubEndpoint(p *packets.PublishPacket, r net.Conn, w net.Conn) error {
	req, err := decodePub(p)
	if err != nil {
		return err
	}

	if err := req.Validate(); err != nil {
		// TODO: log error
		return err
	}

	return session.s.Publish(req.msg)
}

func decodeConn(p *packets.ConnectPacket) connectReq {
	return connectReq{
		password: string(p.Password),
	}
}

func decodeSub(p *packets.SubscribePacket) (subscribeReq, error) {
	if len(p.Topics) != 1 {
		// TODO: handle error
		return subscribeReq{}, mqtt.ErrMalformedData
	}
	chanID, err := getChanID(p.Topics[0])
	if err != nil {
		return subscribeReq{}, mqtt.ErrMalformedData
	}

	return subscribeReq{
		chanID: strconv.FormatUint(chanID, 10),
	}, nil
}

func decodePub(p *packets.PublishPacket) (publishReq, error) {
	chanID, err := getChanID(p.TopicName)
	if err != nil {
		return publishReq{}, mqtt.ErrMalformedData
	}

	msg := mainflux.RawMessage{
		Channel:  chanID,
		Protocol: protocol,
		Payload:  p.Payload,
	}

	return publishReq{msg: msg}, nil
}

func encodeConn(res connectRes, w net.Conn) error {
	if res.err != nil {
		cp := &packets.ConnackPacket{}
		switch res.err {
		case errBadUsernameOrPassword:
			cp.ReturnCode = packets.ErrRefusedBadUsernameOrPassword
		case mqtt.ErrUnauthorized:
			cp.ReturnCode = packets.ErrRefusedNotAuthorised
		default:
			return nil
		}
		if err := cp.Write(w); err != nil {
			// TODO: log error
			return err
		}
	}

	return nil
}

func encodeSub(res subscribeRes, w net.Conn) error {
	if res.err != nil {
		codes := make([]byte, res.topicsNum)
		for i := range codes {
			codes[i] = 128
		}
		sp := &packets.SubackPacket{
			MessageID:   res.id,
			ReturnCodes: codes,
		}
		return sp.Write(w)
	}

	return nil
}

func getChanID(topic string) (uint64, error) {
	chanID := strings.TrimPrefix(topic, prefix)
	cid, err := strconv.ParseUint(chanID, 10, 64)
	if err != nil {
		return 0, mqtt.ErrMalformedData
	}

	return cid, nil
}

func receiveMessages(msgCh chan mainflux.RawMessage, sp *packets.SubscribePacket, w net.Conn) {
	for rawMsg := range msgCh {
		data, err := proto.Marshal(&rawMsg)
		if err != nil {
			// TODO: log error
		}
		p := &packets.PublishPacket{
			TopicName: fmt.Sprintf("channel/%d", rawMsg.Channel),
			Payload:   data,
		}
		if err := p.Write(w); err != nil {
			// TODO: log error
		}
	}
}
