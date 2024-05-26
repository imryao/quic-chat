package chat

import (
	"encoding/json"
	"io"
)

type Message struct {
	Version string `json:"version"`
	Kind    string `json:"kind"`
	From    string `json:"from"`
	To      string `json:"to"`
	ID      string `json:"id"`
	Data    string `json:"data"`
}

func (m *Message) Read(r io.Reader) error {
	return json.NewDecoder(r).Decode(m)
}

func (m *Message) Write(w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}
