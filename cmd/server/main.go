package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"quic-chat/internal/chat"
	"quic-chat/internal/logging"

	"github.com/rs/zerolog/log"

	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	_ = logging.Init()
	if err := run(); err != nil {
		log.Warn().Err(err).Msg("run error")
		os.Exit(1)
	}
}

func run() error {
	serverName := flag.String("n", "", "server name")
	quicPort := flag.Int("q", 4242, "quic port")
	httpPort := flag.Int("h", 9080, "http port")
	bufferSize := flag.Int("b", 16, "message buffer size")
	certFile := flag.String("c", "server.crt", "certificate file")
	keyFile := flag.String("k", "server.key", "private key file")
	flag.Parse()

	if *serverName == "" {
		return errors.New("server name is empty")
	}

	server, err := chat.NewServer(*serverName, *quicPort, *httpPort, *bufferSize, *certFile, *keyFile)
	if err != nil {
		return err
	}
	defer func() { _ = server.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Accept(ctx)
	go server.Deliver(ctx)
	go server.Serve()

	log.Info().Str("server_name", *serverName).Int("quic_port", *quicPort).Int("http_port", *httpPort).Int("buffer_size", *bufferSize).Msg("server started")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs

	log.Info().Msg("server shutting down gracefully")

	return nil
}
