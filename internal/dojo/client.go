package dojo

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ProductTypeScanioRepo = "SCANIO-REPO"
)

type Client struct {
	httpc *resty.Client
	url   string
}

func New(url string, token string) Client {
	httpc := resty.New()
	httpc.SetBaseURL(url)
	httpc.SetHeader("Authorization", fmt.Sprintf("Token %s", token))

	return Client{
		httpc: httpc,
		url:   url,
	}
}

type ProductType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GetProductTypesResult struct {
	Count   int           `json:"count"`
	Results []ProductType `json:"results"`
}

type Product struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GetProductsResult struct {
	Count   int       `json:"count"`
	Results []Product `json:"results"`
}

type Engagement struct {
	ID        int `json:"id"`
	ProductID int `json:"product"`
}

type SLAConfiguration struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Critical    int    `json:"critical"`
	High        int    `json:"high"`
	Medium      int    `json:"medium"`
	Low         int    `json:"low"`
}

func GetDefaultSLAConfigurationParams() SLAConfiguration {
	return SLAConfiguration{
		Name:        "SCANIO-SLA-CONFIGURATION",
		Description: "Default scanio sla configuration",
		Critical:    7,
		High:        30,
		Medium:      90,
		Low:         120,
	}
}

type GetSLAConfigurationResult struct {
	Count   int                `json:"count"`
	Results []SLAConfiguration `json:"results"`
}

func (c Client) IsAnySLAConfiguration() (bool, error) {
	var r GetSLAConfigurationResult
	resp, err := c.httpc.R().
		SetResult(&r).
		Get("/api/v2/sla_configurations/")
	if err != nil {
		return false, err
	}
	if resp.StatusCode() != http.StatusOK {
		return false, fmt.Errorf("%d on getting sla_configurations", resp.StatusCode())
	}
	return r.Count > 0, nil
}

func (c Client) GetSLAConfiguration(name string) (*SLAConfiguration, error) {
	var r GetSLAConfigurationResult
	resp, err := c.httpc.R().
		SetResult(&r).
		Get("/api/v2/sla_configurations/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%d on getting sla_configurations '%s'", resp.StatusCode(), name)
	}

	for _, sla := range r.Results {
		if sla.Name == name {
			return &sla, nil
		}
	}
	return nil, fmt.Errorf("sla_configurations with name '%s' was not found", name)
}

func (c Client) CreateSLAConfiguration(sla SLAConfiguration) (*SLAConfiguration, error) {
	var createdSLA SLAConfiguration
	resp, err := c.httpc.R().
		SetBody(map[string]interface{}{
			"name":        sla.Name,
			"description": sla.Description,
			"critical":    sla.Critical,
			"high":        sla.High,
			"medium":      sla.Medium,
			"low":         sla.Low,
		}).
		SetResult(&createdSLA).
		Post("/api/v2/sla_configuration/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("response != 201 on creating sla configuration '%s'", sla.Name)
	}
	return &createdSLA, nil
}

func (c Client) GetOrCreateSLAConfiguration(config SLAConfiguration) (*SLAConfiguration, error) {
	sla, err := c.GetSLAConfiguration(config.Name)
	if err != nil {
		return nil, err
	}
	if sla != nil {
		return sla, nil
	}
	return c.CreateSLAConfiguration(config)
}

func (c Client) GetOrCreateDefaultSLAConfiguration() (*SLAConfiguration, error) {
	return c.GetOrCreateSLAConfiguration(GetDefaultSLAConfigurationParams())
}

func (c Client) GetProductType(productTypeName string) (*ProductType, error) {
	var r GetProductTypesResult
	resp, err := c.httpc.R().
		SetQueryParams(map[string]string{
			"name": productTypeName,
		}).
		SetResult(&r).
		Get("/api/v2/product_types/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%d on getting product_types '%s'", resp.StatusCode(), productTypeName)
	}
	if r.Count > 1 {
		return nil, fmt.Errorf("multiple product_types with the same name '%s'", productTypeName)
	}
	if r.Count == 0 {
		return nil, nil
	}
	return &r.Results[0], nil
}

func (c Client) CreateProductType(productTypeName string) (*ProductType, error) {
	var p ProductType
	resp, err := c.httpc.R().
		SetFormData(map[string]string{
			"name": productTypeName,
		}).
		SetResult(&p).
		Post("/api/v2/product_types/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("%d on getting product '%s'", resp.StatusCode(), productTypeName)
	}
	return &p, nil
}

func (c Client) GetOrCreateProductType(productTypeName string) (*ProductType, error) {
	productType, err := c.GetProductType(productTypeName)
	if err != nil {
		return nil, err
	}
	if productType != nil {
		return productType, nil
	}
	return c.CreateProductType(productTypeName)
}

func (c Client) GetProduct(productName string) (*Product, error) {
	var r GetProductsResult
	resp, err := c.httpc.R().
		SetQueryParams(map[string]string{
			"name": productName,
		}).
		SetResult(&r).
		Get("/api/v2/products/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%d on getting product '%s'", resp.StatusCode(), productName)
	}
	if r.Count > 1 {
		return nil, fmt.Errorf("multiple products with the same name '%s'", productName)
	}
	if r.Count == 0 {
		return nil, nil
	}
	return &r.Results[0], nil
}

func (c Client) CreateProduct(productName string, productType ProductType) (*Product, error) {
	var p Product
	resp, err := c.httpc.R().
		SetFormData(map[string]string{
			"name":        productName,
			"description": fmt.Sprintf("Default desctiption for product: '%s'", productName),
			"prod_type":   strconv.Itoa(productType.ID),
		}).
		SetResult(&p).
		Post("/api/v2/products/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("%d on creating product '%s'", resp.StatusCode(), productName)
	}
	return &p, nil
}

func (c Client) GetOrCreateProduct(productName string, productType ProductType) (*Product, error) {
	product, err := c.GetProduct(productName)
	if err != nil {
		return nil, err
	}
	if product != nil {
		return product, nil
	}
	return c.CreateProduct(productName, productType)
}

func (c Client) CreateEngagement(product Product) (*Engagement, error) {
	var engagement Engagement
	currentDate := time.Now().Format("2006-01-02")
	resp, err := c.httpc.R().
		SetFormData(map[string]string{
			"target_start": currentDate,
			"target_end":   currentDate,
			"status":       "Completed",
			"product":      strconv.Itoa(product.ID),
			"name":         "scan-io",
		}).
		SetResult(&engagement).
		Post("/api/v2/engagements/")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("%d on creating engagement for product '%s'", resp.StatusCode(), product.Name)
	}
	return &engagement, nil
}

func (c Client) ImportScanResult(engagement Engagement, resultPath string, scanType string) error {
	resp, err := c.httpc.R().
		SetFiles(map[string]string{
			"file": resultPath,
		}).
		SetFormData(map[string]string{
			"engagement": strconv.Itoa(engagement.ID),
			"scan_type":  scanType, // "SARIF",
			// "service":    repo.RepoName,
		}).
		Post("/api/v2/import-scan/")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("%d on importing results to engagement '%d'", resp.StatusCode(), engagement.ID)
	}
	return nil
}
