package controllers

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type envelope map[string]interface{}

func (api *searchAPI) logError(r *http.Request, err error) {
	api.log.Error("internal server error", zap.Error(err), zap.String("request_method", r.Method),
		zap.String("request_uri", r.URL.String()))
}

// errorResponse method for sending JSON-formatted error messages to the client with a given status code.
func (api *searchAPI) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{},
) {
	env := envelope{"error": map[string]string{
		"code":    http.StatusText(status),
		"message": message.(string),
	}}

	err := api.writeJSON(w, status, env, nil)
	if err != nil {
		api.logError(r, err)
		w.WriteHeader(500)
	}
}

func (api *searchAPI) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	api.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	api.errorResponse(w, r, 500, message)
}

func (api *searchAPI) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	api.errorResponse(w, r, http.StatusNotFound, message)
}

func (api *searchAPI) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported this resource", r.Method)
	api.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (api *searchAPI) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	api.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (api *searchAPI) EditConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	api.errorResponse(w, r, http.StatusConflict, message)
}

func (api *searchAPI) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limited exceeded"
	api.errorResponse(w, r, http.StatusTooManyRequests, message)
}

func (api *searchAPI) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	api.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (api *searchAPI) InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	api.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (api *searchAPI) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	api.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (api *searchAPI) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	api.errorResponse(w, r, http.StatusForbidden, message)
}
