package main

import (
	"os"
	"osm-search/pkg/di"
	myHttp "osm-search/pkg/http"

	"go.uber.org/zap"
)

func main() {
	service, cleanup, err := di.InitializeSearcherService()
	if err != nil {
		panic(err)
	}

	signal := myHttp.GracefulShutdown()
	cleanup()

	service.Log.Info("Search Server Stopped", zap.String("signal", signal.String()))

	os.Exit(143)
}
