package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ensureContext(ctx context.Context) error {
	if ctx == nil {
		panic("nil context")
	}
	return ctx.Err()
}

func doJSON(ctx context.Context, client *http.Client, method, url string, body any) (status int, respBody []byte, _ error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response body: %w", readErr)
	}

	return resp.StatusCode, b, nil
}

func doJSONAndExpectStatus(
	ctx context.Context,
	client *http.Client,
	method string,
	url string,
	body any,
	okStatus int,
	requestPrefix string,
	errorPrefix string,
) error {
	status, respBody, err := doJSON(ctx, client, method, url, body)
	if err != nil {
		return fmt.Errorf("%s request: %w", requestPrefix, err)
	}
	if status != okStatus {
		return fmt.Errorf("%s error: status=%d, body=%s", errorPrefix, status, string(respBody))
	}
	return nil
}
