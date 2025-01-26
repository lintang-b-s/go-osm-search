package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	helper "osm-search/pkg/http/http-router/router-helper"
	"regexp"

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
	group.POST("/search", api.search)
	group.POST("/autocomplete", api.autocomplete)
}

type searchRequest struct {
	Query string `json:"query" validate:"required"`
	TopK  int    `json:"top_k" validate:"required,min=1,max=100"`
}

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

	results, err := api.searchService.Search(request.Query, request.TopK)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": results}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
	}
}

type autoCompleteRequest struct {
	Query string `json:"query" validate:"required"`
}

func (api *searchAPI) autocomplete(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var request autoCompleteRequest
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

	results, err := api.searchService.Autocomplete(request.Query)
	if err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": results}, headers); err != nil {
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
