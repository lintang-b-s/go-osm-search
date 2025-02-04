package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	helper "github.com/lintang-b-s/osm-search/pkg/http/http-router/router-helper"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/julienschmidt/httprouter"

	"go.uber.org/zap"
)

var (
	regexSearch = regexp.MustCompile("^[A-Za-z0-9_ +,.()]+$")
)

type searchAPI struct {
	searchService SearchService
	log           *zap.Logger
}

func New(searchService SearchService, log *zap.Logger) *searchAPI {
	return &searchAPI{
		searchService: searchService,
		log:           log,
	}

}

func (api *searchAPI) Routes(group *helper.RouteGroup) {
	group.GET("/search", api.search)
	group.GET("/autocomplete", api.autocomplete)
	group.GET("/reverse", api.reverseGeocoding)
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
	Query  string `json:"query" validate:"required"`               // query entered by the user.
	TopK   int    `json:"top_k" validate:"required,min=1,max=100"` // the number of relevant documents you want to display in the full text search results.
	Offset int    `json:"offset" validate:"min=0"`                 // offset for pagination
}

// searchResponse model info
//
//	@Description	response body untuk hasil full text search.
type searchResponse struct {
	Data []datastructure.Node `json:"data"` // list of osm objects that are relevant to the query given by the user.
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
func (api *searchAPI) search(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request searchRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
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
	} else if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query must be alphanumeric or contain special characters: +, ., (, ), ,"))
		return
	}

	results, err := api.searchService.Search(request.Query, request.TopK, request.Offset)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": results}, headers); err != nil {
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
	var request searchRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
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
	} else if !notMatch {
		api.BadRequestResponse(w, r, fmt.Errorf("validation error: "+"query must be alphanumeric or contain special characters: +, ., (, ), ,"))
		return
	}

	results, err := api.searchService.Autocomplete(request.Query, request.TopK, request.Offset)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": results}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type reverseGeocodingRequest struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

type reverseGeocodingResponse struct {
	Data datastructure.Node `json:"data"`
}

// reverseGeocoding godoc
// @Summary		reverseGeocoding operation allows users to get nearest osm objects based on the latitude and longitude given by the user.
// @Description	reverseGeocoding operation allows users to get nearest osm objects based on the latitude and longitude given by the user.
// @Tags			search
// @ID reverse-geocoding
// @Param			body	body	reverseGeocodingRequest	true
// @Accept			application/json
// @Produce		application/json
// @Router			/api/reverse [get]
// @Success		200	{object}	reverseGeocodingResponse
// @Failure		400	{object}	errorResponse
// @Failure		500	{object}	errorResponse
func (api *searchAPI) reverseGeocoding(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request reverseGeocodingRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
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

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": result}, headers); err != nil {
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
