package wp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	userAgent    = "Postulator/2"
	maxBodyBytes = 32 << 20
)

type request struct {
	header       http.Header
	query        url.Values
	client       *http.Client
	method       string
	namespace    string
	path         string
	contentType  string
	body         []byte
	keepRedirect bool
}

type attempt struct {
	resp   *http.Response
	body   []byte
	err    error
	status int
}

func (c *Client) do(ctx context.Context, req request) (*http.Response, []byte, error) {
	target := c.resolve(req.namespace, req.path, req.query)
	client := req.client
	if client == nil {
		client = c.http
	}

	var last error
	for try := 0; try <= c.retries; try++ {
		if try > 0 {
			if err := c.pause(ctx, delayFor(last, c.backoff(try-1))); err != nil {
				return nil, nil, err
			}
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, nil, errors.New(errors.Cancelled, "the WordPress request was cancelled while waiting for the site rate limit").WithInternal(err)
		}

		started := time.Now()
		result := c.send(ctx, client, req, target)
		c.logAttempt(req, result, time.Since(started))

		if result.err == nil {
			return result.resp, result.body, nil
		}
		if !retryable(result.err) {
			return nil, nil, result.err
		}
		last = result.err
	}
	return nil, nil, last
}

func (c *Client) send(ctx context.Context, client *http.Client, req request, target string) attempt {
	var reader io.Reader
	if len(req.body) > 0 {
		reader = bytes.NewReader(req.body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, target, reader)
	if err != nil {
		return attempt{err: errors.New(errors.Invalid, "the WordPress request could not be built").WithInternal(err)}
	}

	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)
	if req.contentType != "" {
		httpReq.Header.Set("Content-Type", req.contentType)
	}
	for key, values := range req.header {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return attempt{err: transportError(ctx, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return attempt{status: resp.StatusCode, err: transportError(ctx, err)}
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		if req.keepRedirect && resp.StatusCode < http.StatusBadRequest {
			return attempt{resp: resp, body: body, status: resp.StatusCode}
		}
		return attempt{status: resp.StatusCode, err: classify(resp, body)}
	}
	return attempt{resp: resp, body: body, status: resp.StatusCode}
}

func (c *Client) pause(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return errors.New(errors.Cancelled, "the WordPress request was cancelled before it could be retried").WithInternal(ctx.Err())
	case <-timer.C:
		return nil
	}
}

func (c *Client) logAttempt(req request, result attempt, took time.Duration) {
	fields := []zap.Field{
		zap.String("method", req.method),
		zap.String("path", req.namespace+req.path),
		zap.Int("status", result.status),
		zap.Int64("durationMs", took.Milliseconds()),
	}

	if result.err == nil {
		c.logger.Debug("wordpress request", fields...)
		return
	}

	fields = append(fields, zap.String("code", errors.CodeOf(result.err).String()))
	if wpCode := detailString(result.err, "code"); wpCode != "" {
		fields = append(fields, zap.String("wpCode", wpCode))
	}
	c.logger.Warn("wordpress request failed", fields...)
}

func decodeJSON(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return errors.New(errors.External, "the WordPress response is not valid JSON").WithInternal(err)
	}
	return nil
}
