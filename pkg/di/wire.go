//go:build wireinject

//go:generate wire
package di

import (
	"context"
	"osm-search/pkg/di/config"
	shortcontext "osm-search/pkg/di/context"
	kv_di "osm-search/pkg/di/kv"
	logger_di "osm-search/pkg/di/logger"
	searcher_di "osm-search/pkg/di/searcher"
	searchHttp "osm-search/pkg/http"
	"osm-search/pkg/http/http-router/controllers"
	"osm-search/pkg/http/usecases"

	"github.com/google/wire"
	"go.uber.org/zap"
)

var defaultSet = wire.NewSet(
	shortcontext.New,
	config.New,
	logger_di.New,
	kv_di.New,
	searcher_di.New,
)

var searcherSet = wire.NewSet(
	defaultSet,
	NewSearcherService,
	NewSearchAPIServer,
)

func NewSearcherService(log *zap.Logger, searcher usecases.Searcher) controllers.SearchService {
	return usecases.New(log, searcher)
}

func NewSearchAPIServer(ctx context.Context, log *zap.Logger,
	searchService controllers.SearchService) (*searchHttp.Server, error) {
	api := searchHttp.NewServer(log)

	apiService, err := api.Use(
		ctx, log, searchService,
	)
	if err != nil {
		return nil, err
	}

	return apiService, nil
}

func InitializeSearcherService() (*searchHttp.Server, func(), error) {

	panic(wire.Build(searcherSet))
}
