package controllers

import (
	"net/http"
	"net/url"

	"github.com/quero-edu/loki-rule-operator/internal/logger"
)

func ValidateLogQLOnServerFunc(client *http.Client, lokiURL string, logger logger.Logger, logQLExpr string) (bool, error) {
	logQLExprEscaped := url.QueryEscape(logQLExpr)
	lokiQueryEndpoint := "/loki/api/v1/query?query=" + logQLExprEscaped
	logQLURIWithQuery := lokiURL + lokiQueryEndpoint

	response, err := client.Get(logQLURIWithQuery)
	if err != nil {
		logger.Error(err, "Failed to send request to Loki server")
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return true, nil
	}

	logger.Warn("The query string:", logQLExpr, "is not a valid LogQL query")
	return false, nil
}
