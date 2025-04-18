package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geofence"
	helper "github.com/lintang-b-s/osm-search/pkg/http/http-router/router-helper"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/julienschmidt/httprouter"

	"go.uber.org/zap"
)

var (
	regexSearch     = regexp.MustCompile("^[A-Za-z0-9_ +,.()]+$")
	regexOSMFeature = regexp.MustCompile("^[a-zA-Z0-9_:=]+$")
	regexFenceName  = regexp.MustCompile("^[A-Za-z0-9_]+$")
)

type searchAPI struct {
	searchService   SearchService
	geofenceService GeofenceService
	log             *zap.Logger
}

func New(searchService SearchService, geofenceService GeofenceService, log *zap.Logger) *searchAPI {
	return &searchAPI{
		searchService:   searchService,
		log:             log,
		geofenceService: geofenceService,
	}

}

func (api *searchAPI) Routes(group *helper.RouteGroup) {
	group.GET("/search", api.search)
	group.GET("/autocomplete", api.autocomplete)
	group.GET("/reverse", api.reverseGeocoding)
	group.GET("/places", api.nearbyPlaces)
	// geofences
	group.POST("/geofence", api.addGeofence)
	group.DELETE("/geofence/:fencename", api.deleteGeofence)
	group.PUT("/geofence/:fencename/point", api.setQueryPoint)
	group.GET("/geofence/:fencename", api.searchFence)
	group.PUT("/geofence/:fencename", api.addFencePoint)
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// searchRequest model info
//
//	@Description	request body for full text search.
type searchRequest struct {
	Query  string  `json:"query" validate:"required"`                // query entered by the user.
	TopK   int     `json:"top_k" validate:"required,min=1,max=100"`  // the number of relevant documents you want to display in the full text search results.
	Offset int     `json:"offset" validate:"min=0"`                  // offset for pagination
	Lat    float64 `json:"lat" validate:"required,min=-90,max=90"`   // latitude of the user.
	Lon    float64 `json:"lon" validate:"required,min=-180,max=180"` // longitude of the user.
}

// searchResponse model info
//
//	@Description	response body untuk hasil full text search.
type searchResponse struct {
	Place    datastructure.Node `json:"osm_object"`
	Distance float64            `json:"distance"`
}

func NewSearchResponse(data []datastructure.Node, dists []float64) []searchResponse {
	response := make([]searchResponse, 0, len(data))

	for i, d := range data {
		response = append(response, searchResponse{
			Place:    d,
			Distance: dists[i],
		})
	}
	return response
}

// search godoc
// @Summary		search operation to find osm objects relevant to the query given by the user. Support spelling correction.
// @Description	search operation to find osm objects relevant to the query given by the user. Support spelling correction.
// @Tags			search
// @ID search
// @Param			body	body	searchRequest	true
// @Accept			application/json
// @Produce		application/json
// @Router			/api/search [get]
// @Success		200	{object}	searchResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) search(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var (
		request searchRequest
		err     error
	)
	query := r.URL.Query()
	request.Query = query.Get("query")
	
	request.TopK, err = strconv.Atoi(query.Get("top_k"))
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Offset, err = strconv.Atoi(query.Get("offset"))
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Lat, err = strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Lon, err = strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}

	
	validate := validator.New()
	notMatch := regexSearch.MatchString(request.Query)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query must be alphanumeric or contain special characters: +, ., (, ), ,"))
		return
	}

	results, err := api.searchService.Search(request.Query, request.TopK, request.Offset)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	dists := make([]float64, len(results))
	for i, r := range results {
		dists[i] = datastructure.HaversineDistance(request.Lat, request.Lon, r.Lat, r.Lon)
	}

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewSearchResponse(results, dists)}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

// autocomplete godoc
// @Summary		autocomplete operation allows users to search for osm objects based on the prefix of the query.
// @Description	autocomplete operation allows users to search for osm objects based on the prefix of the query.
// @Tags			search
// @ID autocomplete
// @Param			body	body	searchRequest	true
// @Accept			application/json
// @Produce		application/json
// @Router			/api/autocomplete [get]
// @Success		200	{object}	searchResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) autocomplete(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		request searchRequest
		err     error
	)
	query := r.URL.Query()
	request.Query = query.Get("query")
	
	request.TopK, err = strconv.Atoi(query.Get("top_k"))
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Offset, err = strconv.Atoi(query.Get("offset"))
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Lat, err = strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}
	request.Lon, err = strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("top_k must be an integer"))
		return
	}

	validate := validator.New()
	notMatch := regexSearch.MatchString(request.Query)
	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query must be alphanumeric or contain special characters: +, ., (, ), ,"))
		return
	}

	results, err := api.searchService.Autocomplete(request.Query, request.TopK, request.Offset)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	dists := make([]float64, len(results))
	for i, r := range results {
		dists[i] = datastructure.HaversineDistance(request.Lat, request.Lon, r.Lat, r.Lon)
	}

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewSearchResponse(results, dists)}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type reverseGeocodingRequest struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

type reverseGeocodingResponse struct {
	Data datastructure.Node `json:"data"`
	Dist float64            `json:"dist"`
}

func NewReverseGeocodingResponse(data datastructure.Node, dist float64) reverseGeocodingResponse {
	return reverseGeocodingResponse{
		Data: data,
		Dist: dist,
	}
}

// reverseGeocoding godoc
// @Summary		reverseGeocoding operation allows users to get nearest osm objects based on the latitude and longitude given by the user.
// @Description	reverseGeocoding operation allows users to get nearest osm objects based on the latitude and longitude given by the user.
// @Tags			search
// @ID reverse-geocoding
// @Param        lat	query	float	true	"Latitude"
// @Param        lon	query	float	true	"Longitude"
// @Accept			application/json
// @Produce		application/json
// @Router			/api/reverse [get]
// @Success		200	{object}	reverseGeocodingResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) reverseGeocoding(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	query := r.URL.Query()
	if query.Get("lat") == "" || query.Get("lon") == "" {
		api.BadRequestResponse(w, r, errors.New("lat and lon must be provided"))
		return
	}

	lat, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}
	lon, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}
	request := reverseGeocodingRequest{
		Lat: lat,
		Lon: lon,
	}

	validate := validator.New()
	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}

	result, err := api.searchService.ReverseGeocoding(request.Lat, request.Lon)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewReverseGeocodingResponse(result, datastructure.HaversineDistance(
		request.Lat, request.Lon, result.Lat, result.Lon,
	))}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type nearbyPlacesRequest struct {
	Lat     float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon     float64 `json:"lon" validate:"required,min=-180,max=180"`
	Feature string  `json:"feature" validate:"required"`
	Radius  float64 `json:"radius" validate:"max=1000"`
	K       int     `json:"k" validate:"required,min=1,max=100"`
	Offset  int     `json:"offset" validate:"min=0"`
}

// nearbyPlaces godoc
// @Summary		nearbyPlaces operation allows users to get nearest osm objects based on the latitude and longitude given by the user within a certain radius with specific feature.
// @Description	nearbyPlaces operation allows users to get nearest osm objects based on the latitude and longitude given by the user within a certain radius with specific feature.
// @Tags			search
// @ID nearest-places
// @Param        lat	query	float	true	"Latitude"
// @Param        lon	query	float	true	"Longitude"
// @Param        feature	query	string	true	"Feature"
// @Param        radius	float	false	"Radius"
// @Param        k	query	int	true	"total nearest places"
// @Param        offset	query	int	false	"offset"
// @Accept			application/json
// @Produce		application/json
// @Router			/api/places [get]
// @Success		200	{object}	searchResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) nearbyPlaces(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	query := r.URL.Query()
	if query.Get("lat") == "" || query.Get("lon") == "" || query.Get("feature") == "" || query.Get("k") == "" {
		api.BadRequestResponse(w, r, errors.New("lat, lon, feature, and k must be provided"))
		return
	}

	lat, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("lat must be a float"))
		return
	}
	lon, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("lon must be a float"))
		return
	}

	feature := query.Get("feature")
	if !regexOSMFeature.MatchString(feature) {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"feature must be alphanumeric or contain special characters: =, :"))
		return
	}

	radius := 5.0
	if query.Get("radius") != "" {
		radius, err = strconv.ParseFloat(query.Get("radius"), 64)
		if err != nil {
			api.BadRequestResponse(w, r, errors.New("radius must be a float"))
			return
		}
	}

	k, err := strconv.Atoi(query.Get("k"))

	if err != nil {
		api.BadRequestResponse(w, r, errors.New("k must be an integer"))
		return
	}

	offset := 0

	if query.Get("offset") != "" {
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil {
			api.BadRequestResponse(w, r, errors.New("offset must be an integer"))
			return
		}
	}

	request := nearbyPlacesRequest{
		Lat:     lat,
		Lon:     lon,
		Feature: feature,
		Radius:  radius,
		K:       k,
		Offset:  offset,
	}

	validate := validator.New()
	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}

	results, err := api.searchService.NearestNeighboursRadiusWithFeatureFilter(request.K, request.Offset, request.Lat, request.Lon,
		request.Radius, request.Feature)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	dists := make([]float64, len(results))
	for i, r := range results {
		dists[i] = datastructure.HaversineDistance(request.Lat, request.Lon, r.Lat, r.Lon)
	}

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewSearchResponse(results, dists)}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type addGeofenceRequest struct {
	FenceName string `json:"fence_name" validate:"required"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func NewMessageResponse(msg string) messageResponse {
	return messageResponse{msg}
}

// addGeofence godoc
// @Summary		addGeofence operation allows user to add geofence object.
// @Description	addGeofence operation allows user to add geofence object.
// @Tags			search
// @ID add-geofence
// @Param			body	body	addGeofenceRequest	true
// @Accept			application/json
// @Produce		application/json
// @Router			/api/geofence [post]
// @Success		200	{object}	messageResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) addGeofence(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var request addGeofenceRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	if err := r.Body.Close(); err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	validate := validator.New()
	notMatch := regexFenceName.MatchString(request.FenceName)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fencename must be alphanumeric or contain special characters: _"))
		return
	}

	err = api.geofenceService.AddFence(request.FenceName)
	if err != nil {
		api.getStatusCode(w, r, err)
		return
	}
	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewMessageResponse("add geofence success")}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

// deleteGeofence godoc
// @Summary		deleteGeofence operation allows user to delete geofence object.
// @Description	deleteGeofence operation allows user to delete geofence object.
// @Tags			search
// @ID delete-geofence
// @Param			 fencename	path  string	true	"fencename"
// @Accept			application/json
// @Produce		application/json
// @Router			/api/geofence [delete]
// @Success		200	{object}	messageResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) deleteGeofence(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var request addGeofenceRequest
	request.FenceName = ps.ByName("fencename")

	validate := validator.New()
	notMatch := regexFenceName.MatchString(request.FenceName)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fencename must be alphanumeric or contain special characters: _"))
		return
	}

	api.geofenceService.DeleteFence(request.FenceName)
	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewMessageResponse("delete geofence success")}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type queryPointRequest struct {
	FenceName    string  `json:"fence_name" validate:"required"`
	Lat          float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon          float64 `json:"lon" validate:"required,min=-180,max=180"`
	QueryPointID string  `json:"query_point_id" validate:"required"`
}

// setQueryPoint godoc
// @Summary		setQueryPoint operation allows user to set/update query point.
// @Description	setQueryPoint operation allows user to set/update query point.
// @Tags			search
// @ID set-queryPoint-geofence
// @Param			fencename	path  string	true	"fencename"
// @Accept			application/json
// @Produce		application/json
// @Router			/geofence/:fencename/point [post]
// @Success		200	{object}	messageResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) setQueryPoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var request queryPointRequest
	request.FenceName = ps.ByName("fencename")

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	if err := r.Body.Close(); err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	validate := validator.New()
	notMatch := regexFenceName.MatchString(request.FenceName)
	notMatchQueryPointID := regexFenceName.MatchString(request.QueryPointID)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fencename must be alphanumeric or contain special characters: _"))
		return
	}

	if !notMatchQueryPointID {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query_point_id must be alphanumeric or contain special characters: _"))
		return
	}

	err = api.geofenceService.UpdateFencePoint(request.FenceName, request.Lat, request.Lon, request.QueryPointID)
	if err != nil {
		api.getStatusCode(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewMessageResponse("set query point success")}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type FenceResp struct {
	Status string `json:"fence_status"` // fence - querypoint status
	Fence  struct {
		Key       string  `json:"key"`
		CenterLat float64 `json:"center_lat"`
		CenterLon float64 `json:"center_lon"`
		Radius    float64 `json:"radius"` // in km
	} `json:"fence"`
}

func NewFenceResp(status, key string, lat, lon, radius float64) FenceResp {
	return FenceResp{Status: status, Fence: struct {
		Key       string  `json:"key"`
		CenterLat float64 `json:"center_lat"`
		CenterLon float64 `json:"center_lon"`
		Radius    float64 `json:"radius"` // in km
	}{
		Key:       key,
		CenterLat: lat,
		CenterLon: lon,
		Radius:    radius,
	}}
}

type searchFence struct {
	Fences []FenceResp `json:"fences"`
}

func NewSearchFence(fences []geofence.FenceStatusObj) searchFence {
	fenceResp := make([]FenceResp, len(fences))
	for i := 0; i < len(fences); i++ {
		f := fences[i]
		fenceResp[i] = NewFenceResp(getFenceStatus(f.Status),
			f.Fence.GetKey(), f.Fence.GetCenterLat(), f.Fence.GetCenterLon(),
			f.Fence.GetRadius())
	}
	return searchFence{fenceResp}
}

func getFenceStatus(fenceStatus geofence.FenceStatus) string {
	switch fenceStatus {
	case geofence.INSIDE:
		return "INSIDE"
	case geofence.ENTER:
		return "ENTER"
	case geofence.EXIT:
		return "EXIT"
	case geofence.OUTSIDE:
		return "OUTSIDE"
	case geofence.CROSS:
		return "CROSS"
	}
	return ""
}

// searchFence godoc
// @Summary		searchFence operation allows users to search/check is queryPoint inside geofence.
// @Description	searchFence operation allows users to search/check is queryPoint inside geofence.
// @Tags			search
// @ID search-fence
// @Param        lat	query	float	true	"Latitude"
// @Param        lon	query	float	true	"Longitude"
// @Param        fencename path string true "fencename"
// @Param query_point_id query string true "query_point_id"
// @Accept			application/json
// @Produce		application/json
// @Router			/geofence/:fencename [get]
// @Success		200	{object}	searchResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) searchFence(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	fenceName := ps.ByName("fencename")

	query := r.URL.Query()

	lat, err := strconv.ParseFloat(query.Get("lat"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("lat must be a float"))
		return
	}
	lon, err := strconv.ParseFloat(query.Get("lon"), 64)
	if err != nil {
		api.BadRequestResponse(w, r, errors.New("lon must be a float"))
		return
	}

	queryPointID := query.Get("query_point_id")

	request := queryPointRequest{
		FenceName:    fenceName,
		Lat:          lat,
		Lon:          lon,
		QueryPointID: queryPointID,
	}

	validate := validator.New()
	notMatch := regexFenceName.MatchString(request.FenceName)
	notMatchQueryPointID := regexFenceName.MatchString(request.QueryPointID)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fencename must be alphanumeric or contain special characters: _"))
		return
	}

	if !notMatchQueryPointID {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query_point_id must be alphanumeric or contain special characters: _"))
		return
	}

	results, err := api.geofenceService.Search(request.FenceName, request.Lat, request.Lon, request.QueryPointID)
	if err != nil {
		api.getStatusCode(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewSearchFence(results)}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type addFencePointRequest struct {
	FenceName      string  `json:"fence_name" validate:"required"`
	Lat            float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon            float64 `json:"lon" validate:"required,min=-180,max=180"`
	FencePointName string  `json:"fence_point_name" validate:"required"`
	Radius         float64 `json:"radius" validate:"required,min=0.2,max=20"`
}

func (api *searchAPI) addFencePoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fencename := ps.ByName("fencename")
	var request addFencePointRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	if err := r.Body.Close(); err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	request.FenceName = fencename

	validate := validator.New()
	notMatch := regexFenceName.MatchString(request.FenceName)
	notMatchFencePoint := regexFenceName.MatchString(request.FencePointName)

	if err := validate.Struct(request); err != nil {
		english := en.New()
		uni := ut.New(english, english)
		trans, _ := uni.GetTranslator("en")
		_ = enTranslations.RegisterDefaultTranslations(validate, trans)
		vv := translateError(err, trans)
		vvString := []string{}
		for _, v := range vv {
			vvString = append(vvString, v.Error())
		}
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: %v", vvString))
		return
	}
	if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fence_name must be alphanumeric or contain special characters: _"))
		return
	}

	if !notMatchFencePoint {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"fence_point_name must be alphanumeric or contain special characters: _"))
		return
	}

	err = api.geofenceService.AddFencePoint(request.FenceName, request.FencePointName, request.Lat, request.Lon, request.Radius)
	headers := make(http.Header)
	if err != nil {
		api.getStatusCode(w, r, err)
		return
	}

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewMessageResponse("set fence point success")}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

func translateError(err error, trans ut.Translator) (errs []error) {
	if err == nil {
		return nil
	}
	validatorErrs := err.(validator.ValidationErrors)
	for _, e := range validatorErrs {
		translatedErr := fmt.Errorf(e.Translate(trans))
		errs = append(errs, translatedErr)
	}
	return errs
}

func (api *searchAPI) getStatusCode(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		headers := make(http.Header)

		if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewMessageResponse("success")}, headers); err != nil {
			api.ServerErrorResponse(w, r, err)
		}
	}
	var ierr *pkg.Error
	if !errors.As(err, &ierr) {
		api.ServerErrorResponse(w, r, err)
	} else {
		switch ierr.Code() {
		case pkg.ErrInternalServerError:
			api.ServerErrorResponse(w, r, err)
		case pkg.ErrNotFound:
			api.NotFoundResponse(w, r)
		case pkg.ErrConflict:
			api.EditConflictResponse(w, r)
		case pkg.ErrBadParamInput:
			errMsg := errors.New(err.Error())
			api.BadRequestResponse(w, r, errMsg)
		default:
			api.ServerErrorResponse(w, r, err)
		}
	}
}
