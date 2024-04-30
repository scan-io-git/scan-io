package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
)

type Client struct {
	APIClient *bitbucketv1.APIClient
}

type AuthInfo struct {
	Username string // Username for BB access
	Token    string // Token for basic authentication
}

// NewClient initializes a new Bitbucket v1 API client
func NewClient(VCSURL string, auth AuthInfo) (*Client, context.CancelFunc) {
	baseURL := fmt.Sprintf("https://%s/rest", VCSURL)
	config := bitbucketv1.NewConfiguration(baseURL, func(cfg *bitbucketv1.Configuration) {
		// TODO add config values from yaml file
		cfg.HTTPClient = &http.Client{
			Timeout: time.Second * 30, // Set a timeout for HTTP requests
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	config.AddDefaultHeader("Content-Type", "application/json")
	config.AddDefaultHeader("Accept", "application/json")
	basicAuth := bitbucketv1.BasicAuth{
		UserName: auth.Username,
		Password: auth.Token,
	}

	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, basicAuth)
	apiClient := bitbucketv1.NewAPIClient(ctx, config)

	return &Client{
		APIClient: apiClient,
	}, cancel
}
