/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/scan-io-git/scan-io/internal/dojo"

	"github.com/spf13/cobra"
)

var DOJO_TOKEN = os.Getenv("DEFECTDOJO_TOKEN")

type UploadOptions struct {
	URL         string
	InputFile   string
	ProductName string
	ScanType    string
}

var allUploadOptions UploadOptions

// upload2Cmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload2",
	Short: "[EXPERIMENTAL] Upload results to defectdojo",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("uploadCmd called")
		fmt.Printf("DefectDojo URL: %s\n", allUploadOptions.URL)

		dojoClient := dojo.New(allUploadOptions.URL, DOJO_TOKEN)

		exists, err := dojoClient.IsAnySLAConfiguration()
		if err != nil {
			return err
		}

		if !exists {
			if _, err := dojoClient.CreateSLAConfiguration(dojo.GetDefaultSLAConfigurationParams()); err != nil {
				return err
			}
		}

		productType, err := dojoClient.GetOrCreateProductType(dojo.ProductTypeScanioRepo)
		if err != nil {
			return err
		}

		product, err := dojoClient.GetOrCreateProduct(allUploadOptions.ProductName, *productType)
		if err != nil {
			return err
		}

		engagement, err := dojoClient.CreateEngagement(*product)
		if err != nil {
			return err
		}

		if err = dojoClient.ImportScanResult(*engagement, allUploadOptions.InputFile, allUploadOptions.ScanType); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&allUploadOptions.URL, "url", "u", "http://defectdojo.example.com:8080", "DefectDojo URL")
	uploadCmd.Flags().StringVarP(&allUploadOptions.InputFile, "input", "f", "", "report filepath")
	uploadCmd.Flags().StringVarP(&allUploadOptions.ProductName, "product", "p", "", "product name")
	uploadCmd.Flags().StringVarP(&allUploadOptions.ScanType, "scan-type", "t", "", "scan type")
}
