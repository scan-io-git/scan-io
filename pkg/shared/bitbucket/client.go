package bitbucket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
	"github.com/scan-io-git/scan-io/pkg/shared/config"

	"github.com/hashicorp/go-retryablehttp"
)

type Client struct {
	APIClient *bitbucketv1.APIClient
}

type AuthInfo struct {
	Username string // Username for BB access
	Token    string // Token for basic authentication
}

// NewClient initializes a new Bitbucket v1 API client
func NewClient(VCSURL string, auth AuthInfo, globalConfig *config.Config) (*Client, context.CancelFunc) {
	baseURL := fmt.Sprintf("https://%s/rest", VCSURL)

	cfg := setupBitbucketClientConfiguration(baseURL, &globalConfig.HttpClient)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	cfg.AddDefaultHeader("Content-Type", "application/json")
	cfg.AddDefaultHeader("Accept", "application/json")
	basicAuth := bitbucketv1.BasicAuth{
		UserName: auth.Username,
		Password: auth.Token,
	}

	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, basicAuth)
	apiClient := bitbucketv1.NewAPIClient(ctx, cfg)

	return &Client{
		APIClient: apiClient,
	}, cancel
}

// go-bitbucket-v1 dosen't implement retrying
func setupBitbucketClientConfiguration(baseURL string, httpConfig *config.HttpClient) *bitbucketv1.Configuration {
	// Create a retryable client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = config.SetThen(httpConfig.RetryCount, config.DefaultBitbucketConfig().RetryCount)
	retryClient.RetryWaitMin = config.SetThen(httpConfig.RetryWaitTime, config.DefaultBitbucketConfig().RetryWaitTime)
	retryClient.RetryWaitMax = config.SetThen(httpConfig.RetryMaxWaitTime, config.DefaultBitbucketConfig().RetryMaxWaitTime)

	// Get a standard http.Client with retry logic
	standardClient := retryClient.StandardClient()
	standardClient.Timeout = config.SetThen(httpConfig.Timeout, config.DefaultBitbucketConfig().Timeout)

	var proxyFunc func(*http.Request) (*url.URL, error)
	if httpConfig.Proxy.Host != "" && httpConfig.Proxy.Port != "" {
		proxyURL, err := url.Parse(fmt.Sprintf("%s:%s", httpConfig.Proxy.Host, httpConfig.Proxy.Port))
		if err == nil {
			proxyFunc = http.ProxyURL(proxyURL)
		}
	}

	tr := &http.Transport{
		Proxy:           proxyFunc,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !config.GetBoolValue(httpConfig.TlsClientConfig, "Verify", true)},
	}
	standardClient.Transport = tr

	// Setup configuration for go-bitbucket-v1 with the retryable http client
	cfg := bitbucketv1.NewConfiguration(baseURL, func(cfg *bitbucketv1.Configuration) {
		cfg.HTTPClient = standardClient
	})

	return cfg
}
