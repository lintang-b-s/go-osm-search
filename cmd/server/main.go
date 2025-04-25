package main

import (
	"flag"

	"github.com/lintang-b-s/osm-search/pkg/di"
	myHttp "github.com/lintang-b-s/osm-search/pkg/http"
	"github.com/lintang-b-s/osm-search/pkg/searcher"

	_ "github.com/lintang-b-s/osm-search/cmd/server/docs"
	_ "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

var (
	simiiliarityScoring = flag.String("sc", "BM25_FIELD", "similiarity scoring (only 2: BM25_PLUS or TF_IDF_COSINE)")
	useRateLimit        = flag.Bool("ratelimit", false, "use rate limit")
)

//	@title			OSM Search Engine API
//	@version		1.0
//	@description	This is a openstreetmap search engine server.

//	@contact.name	Lintang Birda Saputra
//	@contact.url	_
//	@contact.email	lintang.birda.saputra@mail.ugm.ac.id

//	@license.name	BSD License
//	@license.url	https://opensource.org/license/bsd-2-clause

// @host		localhost:6060
// @BasePath	/api
func main() {
	flag.Parse()
	var searcherScoring searcher.SimiliarityScoring
	switch *simiiliarityScoring {

	case "BM25_PLUS":
		searcherScoring = searcher.BM25_PLUS
	case "TF_IDF_COSINE":
		searcherScoring = searcher.TF_IDF_COSINE
	case "BM25_FIELD":
		searcherScoring = searcher.BM25_FIELD
	default:
		searcherScoring = searcher.BM25_FIELD
	}

	service, cleanup, err := di.InitializeSearcherService(searcherScoring, *useRateLimit)
	defer cleanup()
	if err != nil {
		panic(err)
	}

	signal := myHttp.GracefulShutdown()

	service.Log.Info("Search Server Stopped", zap.String("signal", signal.String()))
}

// swag init
// entah kenapa ga kegenerate route swaggernya
