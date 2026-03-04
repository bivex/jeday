package delivery

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/savsgio/atreugo/v11"
)

func LoggerMiddleware(ctx *atreugo.RequestCtx) error {
	start := time.Now()

	err := ctx.Next()

	stop := time.Since(start)
	statusCode := ctx.Response.StatusCode()

	logger := log.With().
		Int("status", statusCode).
		Str("method", string(ctx.Method())).
		Str("path", string(ctx.Path())).
		Str("ip", ctx.RemoteAddr().String()).
		Dur("latency", stop).
		Logger()

	if err != nil {
		logger.Error().Err(err).Msg("request failed")
	} else if statusCode >= 400 {
		logger.Warn().Msg("request finished with error status")
	} else {
		logger.Info().Msg("request finished")
	}

	return err
}

func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// In development, you might want pretty logging:
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
