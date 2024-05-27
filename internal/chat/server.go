package chat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/rs/zerolog/log"
)

type Server struct {
	// serverName is the name of the server which can be accessed by other servers
	serverName string

	quicListener *quic.Listener
	httpServer   *http.Server
	httpClient   *http.Client

	// conns maps client addr to connection
	conns map[string]quic.Connection
	// addrs maps client name to client addr
	addrs map[string]string
	// names maps client addr to client name
	names map[string]string

	connsMu sync.RWMutex
	addrsMu sync.RWMutex
	namesMu sync.RWMutex

	messages chan Message
}

func NewServer(serverName string, quicPort int, httpPort int, bufferSize int, certFile string, keyFile string) (*Server, error) {
	tlsConf, err := generateTLSConfig(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", quicPort), tlsConf, &quic.Config{
		// todo: avoid magic numbers
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Warn().Err(err).Msg("quic.ListenAddr error")
		return nil, err
	}

	messages := make(chan Message, bufferSize)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var msg Message
		err = json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			log.Warn().Err(err).Msg("json.Unmarshal error")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// todo: msg validation

		messages <- msg
		w.WriteHeader(http.StatusNoContent)
	})

	return &Server{
		serverName: serverName,

		quicListener: listener,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", httpPort),
			Handler: mux,
		},
		httpClient: &http.Client{},

		conns: map[string]quic.Connection{},
		addrs: map[string]string{},
		names: map[string]string{},

		messages: messages,
	}, nil
}

func (s *Server) Close() error {
	close(s.messages)
	return s.quicListener.Close()
}

// Deliver consumes messages from the channel and sends them to the right client / server
func (s *Server) Deliver(ctx context.Context) {
	for {
		select {
		case message := <-s.messages:
			// check if message belongs to me
			if message.toServer == s.serverName {
				// if so, deliver to the specific client
				s.addrsMu.RLock()
				addr, ok := s.addrs[message.toClient]
				s.addrsMu.RUnlock()

				if ok {
					s.connsMu.RLock()
					conn, ok := s.conns[addr]
					s.connsMu.RUnlock()

					if ok {
						go s.sendMessage(conn, &message)
					} else {
						log.Warn().Str("client_name", message.toClient).Msg("client in addrs but absent in conns, that's weird!")
					}
				} else {
					log.Warn().Str("client_name", message.toClient).Msg("client not found")
				}
			} else {
				// if not, send it to the right server
				go s.sendRemoteMessage(&message)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) Accept(ctx context.Context) {
	for {
		conn, err := s.quicListener.Accept(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("quicListener.Accept error")
			return
		}

		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn quic.Connection) {
	defer func() { _ = conn.CloseWithError(serverError, "failed to handle connection") }()

	s.connsMu.Lock()
	s.conns[conn.RemoteAddr().String()] = conn
	s.connsMu.Unlock()

	log.Info().Str("conn_ip", conn.RemoteAddr().String()).Msg("client connected")

	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			log.Warn().Str("conn_ip", conn.RemoteAddr().String()).Err(err).Msg("conn.AcceptStream error")
			s.removeClient(conn.RemoteAddr().String())
			return
		}

		go s.readMessage(stream, conn.RemoteAddr().String())
	}
}

func (s *Server) removeClient(addr string) {
	s.connsMu.Lock()
	if _, ok := s.conns[addr]; ok {
		delete(s.conns, addr)
	}
	s.connsMu.Unlock()

	s.namesMu.Lock()
	name, ok := s.names[addr]
	if ok {
		delete(s.names, addr)
	}
	s.namesMu.Unlock()

	s.addrsMu.Lock()
	if _, ok = s.addrs[name]; ok {
		delete(s.addrs, name)
	}
	s.addrsMu.Unlock()

	log.Info().Str("conn_ip", addr).Msg("client removed")
}

func (s *Server) readMessage(stream quic.Stream, addr string) {
	defer func() { _ = stream.Close() }()

	var message Message
	if err := message.Read(stream); err != nil {
		log.Warn().Str("conn_ip", addr).Err(err).Msg("message.Read error")
		return
	}

	// todo: add registry task to avoid read lock every time
	// read lock every time
	s.addrsMu.RLock()
	_, ok := s.addrs[message.From]
	s.addrsMu.RUnlock()
	if !ok {
		// write lock only once
		s.addrsMu.Lock()
		// double-check to avoid race condition
		if _, ok = s.addrs[message.From]; !ok {
			s.addrs[message.From] = addr
		}
		s.addrsMu.Unlock()
	}

	// read lock every time
	s.namesMu.RLock()
	_, ok = s.names[addr]
	s.namesMu.RUnlock()
	if !ok {
		// write lock only once
		s.namesMu.Lock()
		// double-check to avoid race condition
		if _, ok = s.names[addr]; !ok {
			s.names[addr] = message.From
		}
		s.namesMu.Unlock()
	}

	s.messages <- message
}

func (s *Server) sendMessage(conn quic.Connection, message *Message) {
	stream, err := conn.OpenStream()
	if err != nil {
		log.Warn().Str("conn_ip", conn.RemoteAddr().String()).Err(err).Msg("conn.OpenStream error")
		return
	}
	defer func() { _ = stream.Close() }()

	if err = message.Write(stream); err != nil {
		log.Warn().Str("conn_ip", conn.RemoteAddr().String()).Err(err).Msg("message.Write error")
		return
	}
}

func generateTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Warn().Err(err).Msg("tls.LoadX509KeyPair error")
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{protocol},
	}, nil
}

// Serve handles tasks sent via HTTP
func (s *Server) Serve() {
	err := s.httpServer.ListenAndServe()
	if err != nil {
		log.Warn().Err(err).Msg("httpServer.ListenAndServe error")
	}
}

func (s *Server) sendRemoteMessage(message *Message) {
	b, err := message.WriteBytes()
	if err != nil {
		log.Warn().Err(err).Msg("message.WriteBytes error")
		return
	}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/v1/tasks", message.toServer), bytes.NewReader(b))
	_, err = s.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Msg("httpClient.Do error")
	}
}
