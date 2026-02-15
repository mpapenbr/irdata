package irdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/mpapenbr/irdata/cache"
	"github.com/mpapenbr/irdata/log"
)

type (
	Option        func(*config)
	TokenProvider func() (string, error)
	config        struct {
		ctx   context.Context
		tp    TokenProvider
		cache cache.Cache
	}
	RateLimit struct {
		Limit     int
		Remaining int
		Reset     time.Time
	}

	IrData struct {
		cfg      config
		client   *retryablehttp.Client
		s3Client *retryablehttp.Client
		rlMutex  sync.Mutex
		baseURL  *url.URL
	}
	s3Link struct {
		Link    string    `json:"link"`
		Expires time.Time `json:"expires"`
	}
)

const baseURL = "https://members-ng.iracing.com/data"

var ErrNoTokenProvider = fmt.Errorf("no token provider configured")

func NewIrData(opts ...Option) (*IrData, error) {
	cfg := config{
		ctx:   context.Background(),
		tp:    func() (string, error) { return "", ErrNoTokenProvider },
		cache: cache.NewNoopCache(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	client := retryablehttp.NewClient()
	client.Logger = newCustomLeveledLogger(log.Default().Named("irapi"))
	s3Client := retryablehttp.NewClient()
	s3Client.Logger = newCustomLeveledLogger(log.Default().Named("ir-s3"))
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &IrData{
		cfg:      cfg,
		client:   client,
		s3Client: s3Client, rlMutex: sync.Mutex{}, baseURL: parsedBaseURL,
	}, nil
}

func WithContext(ctx context.Context) Option {
	return func(c *config) {
		c.ctx = ctx
	}
}

func WithTokenProvider(tp TokenProvider) Option {
	return func(c *config) {
		c.tp = tp
	}
}

func WithCache(arg cache.Cache) Option {
	return func(c *config) {
		c.cache = arg
	}
}

func (i *IrData) Get(uri string) ([]byte, error) {
	if b, ok := i.cfg.cache.Get(uri); ok {
		return b, nil
	}
	token, err := i.cfg.tp()
	if err != nil {
		return nil, err
	}

	uriRef, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	reqURL := i.baseURL.ResolveReference(uriRef)

	req, err := retryablehttp.NewRequestWithContext(
		i.cfg.ctx,
		http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Debug("response received",
		log.Int("status", resp.StatusCode),
		log.String("rate-limit", resp.Header.Get("X-RateLimit-Limit")),
		log.String("rate-remaining", resp.Header.Get("X-RateLimit-Remaining")),
		log.String("rate-reset", resp.Header.Get("X-RateLimit-Reset")),
	)
	defer resp.Body.Close()
	rateReset, _ := strconv.ParseFloat(resp.Header.Get("X-RateLimit-Reset"), 64)
	if rateReset > 0 {
		log.Debug("rate limit reset time",
			log.String("reset-time", time.Unix(int64(rateReset), 0).String()))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var s3link s3Link
	if err := json.Unmarshal(body, &s3link); err == nil {
		s3Resp, err := i.s3Client.Get(s3link.Link)
		if err != nil {
			return nil, err
		}
		defer s3Resp.Body.Close()
		if s3Resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code from s3 link: %d", s3Resp.StatusCode)
		}
		body, err = io.ReadAll(s3Resp.Body)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}
