package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/VictoriaMetrics/grafana-logs-datasource/pkg/utils"
)

const (
	health = "/health"
)

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}
	return &Datasource{
		settings:   settings,
		httpClient: cl,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	settings backend.DataSourceInstanceSettings

	httpClient *http.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
	d.httpClient.CloseIdleConnections()
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	var wg sync.WaitGroup
	for _, q := range req.Queries {
		wg.Add(1)
		go func(q backend.DataQuery) {
			defer wg.Done()
			response.Responses[q.RefID] = d.query(ctx, req.PluginContext, q)
		}(q)
	}
	wg.Wait()

	return response, nil
}

func (d *Datasource) query(ctx context.Context, _ backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var q Query
	if err := json.Unmarshal(query.JSON, &q); err != nil {
		err = fmt.Errorf("failed to parse query json: %s", err)
		return newResponseError(err, backend.StatusBadRequest)
	}

	var settings struct {
		HTTPMethod  string `json:"httpMethod"`
		QueryParams string `json:"customQueryParameters"`
	}
	if err := json.Unmarshal(d.settings.JSONData, &settings); err != nil {
		err = fmt.Errorf("failed to parse datasource settings: %w", err)
		return newResponseError(err, backend.StatusBadRequest)
	}
	if settings.HTTPMethod == "" {
		settings.HTTPMethod = http.MethodPost
	}

	reqURL, err := q.getQueryURL(d.settings.URL, settings.QueryParams)
	if err != nil {
		err = fmt.Errorf("failed to create request URL: %w", err)
		return newResponseError(err, backend.StatusBadRequest)
	}

	req, err := http.NewRequestWithContext(ctx, settings.HTTPMethod, reqURL, nil)
	if err != nil {
		err = fmt.Errorf("failed to create new request with context: %w", err)
		return newResponseError(err, backend.StatusBadRequest)
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to make http request: %w", err)
		return newResponseError(err, backend.StatusBadRequest)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.DefaultLogger.Error("failed to close response body", "err", err.Error())
		}
	}()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("got unexpected response status code: %d", resp.StatusCode)
		return newResponseError(err, backend.Status(resp.StatusCode))
	}

	timeFd := data.NewFieldFromFieldType(data.FieldTypeTime, 0)
	timeFd.Name = "Time"

	lineField := data.NewFieldFromFieldType(data.FieldTypeString, 0)
	lineField.Name = "Line"

	labels := data.Labels{}

	dec := json.NewDecoder(resp.Body)

	for dec.More() {
		var r Response
		err := dec.Decode(&r)
		if err != nil {
			return newResponseError(err, backend.StatusInternal)
		}
		log.DefaultLogger.Info("RESOSNE => %#v", r)
		for fieldName, value := range r {
			switch fieldName {
			case messageField:
				lineField.Append(value)
			case timeField:
				getTime, err := utils.GetTime(value)
				if err != nil {
					return newResponseError(err, backend.StatusInternal)
				}
				timeFd.Append(getTime)
			case streamField:
				expr, err := metricsql.Parse(value)
				if err != nil {
					return newResponseError(err, backend.StatusInternal)
				}
				if mExpr, ok := expr.(*metricsql.MetricExpr); ok {
					for _, filters := range mExpr.LabelFilterss {
						for _, filter := range filters {
							labels[filter.Label] = filter.Value
						}
					}
				}
			default:
				labels[fieldName] = value
			}
		}
	}

	frame := data.NewFrame("", timeFd, lineField)

	for fieldName, value := range labels {
		labelsField := data.NewFieldFromFieldType(data.FieldTypeString, 0)
		labelsField.Name = fieldName
		for i := 0; i < q.MaxLines; i++ {
			labelsField.Append(value)
		}
		frame.Fields = append(frame.Fields, labelsField)
	}

	rsp := backend.DataResponse{}
	frame.Meta = &data.FrameMeta{}
	rsp.Frames = append(rsp.Frames, frame)

	return rsp
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", d.settings.URL, health), nil)
	if err != nil {
		return newHealthCheckErrorf("could not create request"), nil
	}
	resp, err := d.httpClient.Do(r)
	if err != nil {
		return newHealthCheckErrorf("request error"), nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.DefaultLogger.Error("check health: failed to close response body", "err", err.Error())
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return newHealthCheckErrorf("got response code %d", resp.StatusCode), nil
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

// newHealthCheckErrorf returns a new *backend.CheckHealthResult with its status set to backend.HealthStatusError
// and the specified message, which is formatted with Sprintf.
func newHealthCheckErrorf(format string, args ...interface{}) *backend.CheckHealthResult {
	return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: fmt.Sprintf(format, args...)}
}

// newResponseError returns a new backend.DataResponse with its status set to backend.DataResponse
// and the specified error message.
func newResponseError(err error, httpStatus backend.Status) backend.DataResponse {
	log.DefaultLogger.Error(err.Error())
	return backend.DataResponse{Status: httpStatus, Error: err}
}

func labelsToRawJson(labels data.Labels) (json.RawMessage, error) {
	// data.Labels when converted to JSON keep the fields sorted
	bytes, err := json.Marshal(labels)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
