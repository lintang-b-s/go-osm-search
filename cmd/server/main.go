package main

import (
	"osm-search/pkg/di"
	myHttp "osm-search/pkg/http"

	_ "github.com/swaggo/http-swagger" // http-swagger middleware
	"go.uber.org/zap"
)

// @title OSM Search Engine API
// @version 1.0
// @description This is a openstreetmap search engine server.

// @contact.name Lintang Birda Saputra
// @contact.url _
// @contact.email lintang.birda.saputra@mail.ugm.ac.id

// @license.name BSD License
// @license.url https://opensource.org/license/bsd-2-clause

// @host localhost
// @BasePath /api
func main() {
	service, cleanup, err := di.InitializeSearcherService()
	defer cleanup()
	if err != nil {

		panic(err)
	}

	signal := myHttp.GracefulShutdown()

	service.Log.Info("Search Server Stopped", zap.String("signal", signal.String()))
}
