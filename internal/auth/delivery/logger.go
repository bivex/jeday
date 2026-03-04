package delivery

import (
	"os"
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
	levelStr := os.Getenv("LOG_LEVEL")
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
}
