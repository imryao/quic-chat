package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"quic-chat/internal/chat"
	"quic-chat/internal/logging"

	"github.com/rs/zerolog/log"
)

func main() {
	_ = logging.Init()
	if err := run(); err != nil {
		log.Warn().Err(err).Msg("run error")
		os.Exit(1)
	}
}

func run() error {
	clientName := flag.String("n", "", "client name")
	serverAddr := flag.String("s", "127.0.0.1:4242", "server address")
	bufferSize := flag.Int("b", 16, "message buffer size")
	flag.Parse()

	if *clientName == "" {
		return errors.New("client name is empty")
	}

	client, err := chat.NewClient(context.Background(), *clientName, *serverAddr, *bufferSize)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, errs := client.Receive(ctx)

	// todo
	go func() {
		for {
			select {
			case msg := <-messages:
				log.Info().Str("from", msg.From).Str("data", msg.Data).Msg("message received")
			case err = <-errs:
				log.Warn().Err(err).Msg("error received")
			}
		}
	}()

	log.Info().Str("client_name", *clientName).Str("server_addr", *serverAddr).Int("buffer_size", *bufferSize).Msg("client started")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs

	log.Info().Msg("client shutting down gracefully")

	return nil
}
