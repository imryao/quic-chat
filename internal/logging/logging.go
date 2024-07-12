package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func Init() error {
	//f, err := os.OpenFile("/var/log/quic-chat/app.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	//if err != nil {
	//	fmt.Println(err)
	//	return err
	//}
	f := os.Stdout
	log.Logger = zerolog.New(f).With().Timestamp().Logger()
	return nil
}
