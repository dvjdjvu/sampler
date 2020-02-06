package client

import (
	"bytes"
	"encoding/json"
	"github.com/djvu/sampler/metadata"
	"net/http"
)

const (
	backendUrl       = "http://localhost/api/v1"
	installationPath = "/telemetry/installation"
	statisticsPath   = "/telemetry/statistics"
	crashPath        = "/telemetry/crash"
	registrationPath = "/license/registration"
	verificationPath = "/license/verification"
)

// BackendClient is used to verify license and to send telemetry
// for analyses (anonymous usage data statistics and crash reports)
type BackendClient struct {
	client http.Client
}

func NewBackendClient() *BackendClient {
	return &BackendClient{
		client: http.Client{},
	}
}

func (c *BackendClient) ReportInstallation(statistics *metadata.Statistics) {

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(statistics)
	if err != nil {
		c.ReportCrash(err.Error(), statistics)
	}

	_, err = sendRequest(backendUrl+installationPath, buf)

	if err != nil {
		c.ReportCrash(err.Error(), statistics)
	}
}

func (c *BackendClient) ReportStatistics(statistics *metadata.Statistics) {

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(statistics)
	if err != nil {
		c.ReportCrash(err.Error(), statistics)
	}

	_, err = sendRequest(backendUrl+statisticsPath, buf)
	if err != nil {
		c.ReportCrash(err.Error(), statistics)
	}
}

func (c *BackendClient) ReportCrash(error string, statistics *metadata.Statistics) {

	req := struct {
		Error      string
		Statistics *metadata.Statistics
	}{
		error,
		statistics,
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		return
	}

	_, _ = sendRequest(backendUrl+crashPath, buf)
}

func sendRequest(url string, body *bytes.Buffer) (resp *http.Response, err error) {
	c := http.DefaultClient
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.Do(req)
}
