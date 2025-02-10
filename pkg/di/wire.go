//go:build wireinject

//go:generate wire
package di

import (
	"context"

	"github.com/google/wire"
	"github.com/lintang-b-s/osm-search/pkg/di/config"
	shortcontext "github.com/lintang-b-s/osm-search/pkg/di/context"
	geofence_di "github.com/lintang-b-s/osm-search/pkg/di/geofence"
	kv_di "github.com/lintang-b-s/osm-search/pkg/di/kv"
	logger_di "github.com/lintang-b-s/osm-search/pkg/di/logger"
	searcher_di "github.com/lintang-b-s/osm-search/pkg/di/searcher"
	searchHttp "github.com/lintang-b-s/osm-search/pkg/http"
	"github.com/lintang-b-s/osm-search/pkg/http/http-router/controllers"
	"github.com/lintang-b-s/osm-search/pkg/http/usecases"
	"github.com/lintang-b-s/osm-search/pkg/searcher"
	"go.uber.org/zap"
)

var defaultSet = wire.NewSet(
	shortcontext.New,
	config.New,
	logger_di.New,
	kv_di.New,
	searcher_di.New,
	geofence_di.New,
)

var searcherSet = wire.NewSet(
	defaultSet,
	NewSearcherService,
	NewGeofenceService,
	NewSearchAPIServer,
)

func NewSearcherService(log *zap.Logger, searcher usecases.Searcher) controllers.SearchService {
	return usecases.New(log, searcher)
}

func NewGeofenceService(geofenceIndex usecases.GeofenceIndex) controllers.GeofenceService {
	return usecases.NewGeofenceService(geofenceIndex)
}

func NewSearchAPIServer(ctx context.Context, log *zap.Logger,
	searchService controllers.SearchService, geofenceService controllers.GeofenceService) (*searchHttp.Server, error) {
	api := searchHttp.NewServer(log)

	apiService, err := api.Use(
		ctx, log, searchService, geofenceService,
	)
	if err != nil {
		return nil, err
	}

	return apiService, nil
}

func InitializeSearcherService(scoring searcher.SimiliarityScoring) (*searchHttp.Server, func(), error) {

	panic(wire.Build(searcherSet))
}
