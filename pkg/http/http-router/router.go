package http_router

import (
	"context"
	"fmt"
	"osm-search/pkg/http/http-router/controllers"
	router_helper "osm-search/pkg/http/http-router/router-helper"
	http_server "osm-search/pkg/http/server"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type API struct {
	log *zap.Logger
}

func NewAPI(log *zap.Logger) *API {
	return &API{log: log}
}

func (api *API) Run(
	ctx context.Context,
	config http_server.Config,
	log *zap.Logger,

	searchService controllers.SearchService,
) error {
	log.Info("Run httprouter API")

	router := httprouter.New()

	corsHandler := cors.New(cors.Options{ //nolint:gocritic // ignore
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, //nolint:mnd // ignore

	})

	group := router_helper.NewRouteGroup(router, "/api")

	searcherRoutes := controllers.New(searchService, log)

	searcherRoutes.Routes(group)

	mainMwChain := alice.New(corsHandler.Handler, EnforceJSONHandler, api.recoverPanic,
		RealIP, Heartbeat("healthz"), Logger(log), Labels).Then(router)

	srv := http_server.New(ctx, mainMwChain, config)
	log.Info(fmt.Sprintf("API run on port %d", config.Port))

	err := srv.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}
