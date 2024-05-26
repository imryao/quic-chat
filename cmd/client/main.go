package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"quic-chat/internal/chat"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	nickname := flag.String("n", "", "nickname")
	addr := flag.String("s", "localhost", "server")
	flag.Parse()

	if *nickname == "" {
		return errors.New("nickname is empty")
	}

	client, err := chat.NewClient(context.Background(), *addr, *nickname)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, errs := client.Receive(ctx)

	go func() {
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return
				}
				log.Println("Received message:", msg.From, msg.Data)
			case err, ok := <-errs:
				if !ok {
					return
				}
				log.Println("Error receiving message:", err)
			}
		}
	}()

	log.Println("client started")
	select {}

	return nil
}
