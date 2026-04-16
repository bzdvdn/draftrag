package draftrag

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func deleteCollectionHTTP(ctx context.Context, client *http.Client, reqURL string, service string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request: %w", service, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// 404 допустим — коллекция уже не существует (идемпотентность).
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s error: status=%d, body=%s", service, resp.StatusCode, string(body))
	}

	return nil
}
