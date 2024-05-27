package chat

import (
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"regexp"
	"strings"
)

type Message struct {
	Version string `json:"version"`
	Kind    string `json:"kind"`
	From    string `json:"from"`
	To      string `json:"to"`
	ID      string `json:"id"`
	Data    string `json:"data"`

	fromServer string
	fromClient string
	toServer   string
	toClient   string
}

var re = regexp.MustCompile(`.+@.+`)

func (m *Message) Read(r io.Reader) error {
	err := json.NewDecoder(r).Decode(m)
	if err != nil {
		log.Warn().Err(err).Msg("json.Unmarshal error")
		return err
	}

	if !re.MatchString(m.From) || !re.MatchString(m.To) {
		log.Warn().Str("from", m.From).Str("to", m.To).Msg("invalid address")
		return errors.New("invalid address")
	}

	from := strings.Split(m.From, "@")
	m.fromServer = from[1]
	m.fromClient = from[0]
	to := strings.Split(m.To, "@")
	m.toServer = to[1]
	m.toClient = to[0]

	return nil
}

func (m *Message) Write(w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}

func (m *Message) WriteBytes() ([]byte, error) {
	return json.Marshal(m)
}
