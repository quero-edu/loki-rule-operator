package controllers

import (
	"net/http"
	"net/url"
)

func ValidateLogQLOnServerFunc(client *http.Client, lokiURL string, logQLExpr string) (bool, error) {
	logQLExprEscaped := url.QueryEscape(logQLExpr)
	lokiQueryEndpoint := "/loki/api/v1/query?query=" + logQLExprEscaped
	logQLURIWithQuery := lokiURL + lokiQueryEndpoint

	response, err := client.Get(logQLURIWithQuery)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
