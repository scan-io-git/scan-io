package httpclient

import (
	"crypto/tls"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-hclog"
)

type (
	RestyConfig struct {
		Debug            bool
		RetryCount       int
		RetryWaitTime    time.Duration
		RetryMaxWaitTime time.Duration
		Timeout          time.Duration
		Certificates     tls.Certificate // not used
		RootCertificate  string          // not used
		TLSClientConfig  tls.Config
		Proxy            string
		Logger           *hclog.Logger
	}
)

func DefaultConfig() *RestyConfig {
	return &RestyConfig{
		Debug:            false,
		RetryCount:       5,
		RetryWaitTime:    (1 * time.Second), // library default is 100ms
		RetryMaxWaitTime: (2 * time.Second), // library default is 2s
		Timeout:          (10 * time.Second),
		Proxy:            "",
		// Logger: *hclog.Logger,
	}
}

func (config *RestyConfig) BuildClient() (*resty.Client, error) {
	client := resty.New()
	client.SetDebug(config.Debug)
	client.SetRetryCount(config.RetryCount)
	client.SetRetryWaitTime(config.RetryWaitTime)
	client.SetRetryMaxWaitTime(config.RetryMaxWaitTime)
	client.SetTimeout(config.Timeout)
	client.SetProxy(config.Proxy)
	client.SetDebug(config.Debug)

	return client, nil
}

func newClient() *resty.Client {
	// parsingConfig()
	// config :=

	// client, err := config.Build()
	// if err != nil {
	// 	xlog.Jupiter().Error("build resty client failed", zap.Error(err))
	// 	return nil, err
	// }

	return nil
}

func GetClient() *resty.Client {
	return newClient()
}
