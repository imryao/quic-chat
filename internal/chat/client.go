package chat

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Client struct {
	// clientName is the name of the client which can be identified by its server
	clientName string
	// conn is the connection to the server
	conn       quic.Connection
	bufferSize int
}

func NewClient(ctx context.Context, clientName, serverAddr string, bufferSize int) (*Client, error) {
	conn, err := quic.DialAddr(ctx, serverAddr, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{protocol},
	}, nil)
	if err != nil {
		log.Warn().Err(err).Msg("quic.DialAddr error")
		return nil, err
	}

	c := &Client{
		clientName: clientName,
		conn:       conn,
		bufferSize: bufferSize,
	}

	message := Message{
		Version: "raat.cs.ac.cn/v1alpha1",
		Kind:    "ClientRegistration",
		From:    fmt.Sprintf("%s@%s", c.clientName, serverAddr),
		To:      fmt.Sprintf("SERVER@%s", serverAddr),
	}
	_ = c.Send(&message)

	return c, nil
}

func (c *Client) Send(message *Message) error {
	stream, err := c.conn.OpenStream()
	if err != nil {
		log.Warn().Err(err).Msg("quic.OpenStream error")
		return err
	}
	defer func() { _ = stream.Close() }()

	return message.Write(stream)
}

func (c *Client) Receive(ctx context.Context) (<-chan Message, <-chan error) {
	messages, errs := make(chan Message, c.bufferSize), make(chan error, c.bufferSize)
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
