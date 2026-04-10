package databases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mittolabs/applad/internal/model"
)

type postgrestResponse struct {
	Body       []byte
	StatusCode int
	Header     http.Header
}

func (s *Service) rowURL(tableName string) string {
	return strings.TrimRight(s.postgrestURL, "/") + "/" + tableName
}

func (s *Service) forwardToPostgREST(ctx context.Context, method, requestURL, schema string, body []byte, extraHeaders map[string]string) (*postgrestResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Prefer", "return=representation")
	if schema != "" {
		request.Header.Set("Content-Profile", schema)
		request.Header.Set("Accept-Profile", schema)
	}
	for key, value := range extraHeaders {
		request.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("postgrest request: %w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	return &postgrestResponse{
		Body:       responseBody,
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
	}, nil
}

func (s *Service) parseRowResponse(body []byte, tableID, databaseID string) (*model.Row, error) {
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err == nil && len(rows) > 0 {
		return mapToRow(rows[0], tableID, databaseID), nil
	}
	var row map[string]interface{}
	if err := json.Unmarshal(body, &row); err == nil {
		return mapToRow(row, tableID, databaseID), nil
	}
	return nil, fmt.Errorf("postgrest returned unreadable row data")
}

func parsePostgRESTTotal(headers http.Header) int {
	contentRange := headers.Get("Content-Range")
	if contentRange == "" {
		return -1
	}
	slashIndex := strings.LastIndex(contentRange, "/")
	if slashIndex < 0 || slashIndex == len(contentRange)-1 {
		return -1
	}
	total, err := strconv.Atoi(contentRange[slashIndex+1:])
	if err != nil {
		return -1
	}
	return total
}