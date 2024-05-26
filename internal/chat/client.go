package chat

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/quic-go/quic-go"
)

type Client struct {
	id   string
	conn quic.Connection
}

const clientBufferSize = 16

func NewClient(ctx context.Context, addr, id string) (*Client, error) {
	conn, err := quic.DialAddr(ctx, fmt.Sprintf("%s:%d", addr, quicPort), &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{protocol},
	}, nil)
	if err != nil {
		return nil, err
	}

	return &Client{id: id, conn: conn}, nil
}

func (c *Client) Send(text string) error {
	stream, err := c.conn.OpenStream()
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	message := Message{From: c.id, Data: text}

	return message.Write(stream)
}

func (c *Client) Receive(ctx context.Context) (<-chan Message, <-chan error) {
	messages, errs := make(chan Message, clientBufferSize), make(chan error, clientBufferSize)
	go func() {
		defer close(messages)
		defer close(errs)
		for {
			stream, err := c.conn.AcceptStream(ctx)
			if err != nil {
				errs <- err
				return
			}

			go c.readStream(stream, messages, errs)
		}
	}()

	return messages, errs
}

func (c *Client) readStream(stream quic.Stream, messages chan<- Message, errs chan<- error) {
	defer func() { _ = stream.Close() }()

	var message Message
	if err := message.Read(stream); err != nil {
		errs <- err
		return
	}

	messages <- message
}
