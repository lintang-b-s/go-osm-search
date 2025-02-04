package http

import (
	"context"

	http_router "github.com/lintang-b-s/osm-search/pkg/http/http-router"
	"github.com/lintang-b-s/osm-search/pkg/http/http-router/controllers"
	http_server "github.com/lintang-b-s/osm-search/pkg/http/server"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	Log *zap.Logger
}

func NewServer(log *zap.Logger) *Server {
	return &Server{Log: log}
}

func (s *Server) Use(
	ctx context.Context,
	log *zap.Logger,

	searchService controllers.SearchService,

) (*Server, error) {
	viper.SetDefault("API_PORT", 6060)

	viper.SetDefault("API_TIMEOUT", "1000s")

	config := http_server.Config{
		Port:    viper.GetInt("API_PORT"),
		Timeout: viper.GetDuration("API_TIMEOUT"),
	}

	server := http_router.NewAPI(log)

	g := errgroup.Group{}

	g.Go(func() error {
		return server.Run(
			ctx, config, log, searchService,
		)
	})

	return s, nil

}
