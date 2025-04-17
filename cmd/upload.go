package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/scan-io-git/scan-io/internal/dojo"
	"github.com/scan-io-git/scan-io/pkg/shared/errors"
	"github.com/scan-io-git/scan-io/pkg/shared/logger"
)

var DOJO_TOKEN = os.Getenv("SCANIO_DEFECTDOJO_TOKEN")

type UploadOptions struct {
	URL         string `json:"url,omitempty"`
	InputFile   string `json:"input_file,omitempty"`
	ProductName string `json:"product_name,omitempty"`
	ScanType    string `json:"scan_type,omitempty"`
}

var allUploadOptions UploadOptions

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "[EXPERIMENTAL] Upload results to defectdojo",
	Long: `CLI wrapper over defectdojo upload functionality.
Make sure that default SLAConfiguration exists, and create if it does not.
Create new type of products in defectdojo: "SCANIO-REPO".
Create product if it's not exists yet.
Create engagement and import results from file.`,
	Hidden: true,
	Example: `  # Upload json results of semgrep:
  scanio upload -u https://defectdojo.example.com -p github.com/juice-shop/juice-shop -i ~/.scanio/results/github.com/juice-shop/juice-shop/semgrep-2023-05-13T11:09:04Z.json -t "Semgrep JSON Report"
  
  # Upload json results of trufflehog:
  scanio upload -u https://defectdojo.example.com -p github.com/juice-shop/juice-shop -i ~/.scanio/results/github.com/juice-shop/juice-shop/trufflehog-2023-05-18T12:20:12Z.json -t "Trufflehog Scan"`,
	RunE: func(cmd *cobra.Command, args []string) error {

		logger := logger.NewLogger(AppConfig, "core-upload")
		logger.Info("DefectDojo", "URL", allUploadOptions.URL)

		dojoClient := dojo.New(allUploadOptions.URL, DOJO_TOKEN)

		exists, err := dojoClient.IsAnySLAConfiguration()
		if err != nil {
			return errors.NewCommandError(allUploadOptions, nil, err, 1)
		}

		if !exists {
			if _, err := dojoClient.CreateSLAConfiguration(dojo.GetDefaultSLAConfigurationParams()); err != nil {
				return errors.NewCommandError(allUploadOptions, nil, err, 1)
			}
		}

		productType, err := dojoClient.GetOrCreateProductType(dojo.ProductTypeScanioRepo)
		if err != nil {
			return errors.NewCommandError(allUploadOptions, nil, err, 1)
		}

		product, err := dojoClient.GetOrCreateProduct(allUploadOptions.ProductName, *productType)
		if err != nil {
			return errors.NewCommandError(allUploadOptions, nil, err, 1)
		}

		engagement, err := dojoClient.CreateEngagement(*product)
		if err != nil {
			return errors.NewCommandError(allUploadOptions, nil, err, 1)
		}

		if err = dojoClient.ImportScanResult(*engagement, allUploadOptions.InputFile, allUploadOptions.ScanType); err != nil {
			return errors.NewCommandError(allUploadOptions, nil, err, 1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&allUploadOptions.URL, "url", "u", "http://defectdojo.example.com:8080", "DefectDojo URL")
	uploadCmd.Flags().StringVarP(&allUploadOptions.InputFile, "input", "i", "", "report filepath")
	uploadCmd.Flags().StringVarP(&allUploadOptions.ProductName, "product", "p", "", "product name")
	uploadCmd.Flags().StringVarP(&allUploadOptions.ScanType, "scan-type", "t", "", "scan type")
}
