/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	utils "github.com/scan-io-git/scan-io/internal/utils"

	// "github.com/scan-io-git/scan-io/internal/vcs"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"
)

var DEFECTDOJO_TOKEN = os.Getenv("DEFECTDOJO_TOKEN")

type UploadResultsOptions struct {
	URL            string
	InputFile      string
	ProjectsPrefix string
	VCSURL         string
}

var allUploadResultsOptions UploadResultsOptions

type DefectDojoProject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GetProductsResult struct {
	Count   int                 `json:"count"`
	Results []DefectDojoProject `json:"results"`
}

type DefectDojoEngagement struct {
	ID        int `json:"id"`
	ProductID int `json:"product"`
}

func collectUniqueDefectDojoProjectNames(reposDetails []shared.RepositoryParams) []string {
	uniqueNamesMap := make(map[string]interface{})
	for _, repo := range reposDetails {
		if repo.Namespace == "" {
			shared.NewLogger("core").Warn("repo does no have a namespace, skipping", "repo", repo)
			continue
		}
		uniqueNamesMap[repo.Namespace] = struct{}{}
	}

	uniqueNames := []string{}
	for name := range uniqueNamesMap {
		uniqueNames = append(uniqueNames, name)
	}

	return uniqueNames
}

func projectNameToTag(projectName string) string {
	r := md5.Sum([]byte(projectName))
	encodedProjectName := hex.EncodeToString(r[:])
	return fmt.Sprintf("scanio_p_%s", encodedProjectName)
}

func namespaceToDefectDojoProjectName(namespace string) string {
	return fmt.Sprintf("%s%s", allUploadResultsOptions.ProjectsPrefix, namespace)
}

func ensureDefectDojoProjectExistance(reposDetails []shared.RepositoryParams) error {
	client := resty.New()
	client.SetHostURL(allUploadResultsOptions.URL)
	client.SetHeader("Authorization", fmt.Sprintf("Token %s", DEFECTDOJO_TOKEN))

	uniqueNamespaces := collectUniqueDefectDojoProjectNames(reposDetails)

	for _, namespace := range uniqueNamespaces {

		projectName := namespaceToDefectDojoProjectName(namespace)
		projectNameTag := projectNameToTag(projectName)

		var getProductsResult GetProductsResult
		resp, err := client.R().
			SetQueryParams(map[string]string{
				"tag": projectNameTag,
			}).
			SetResult(&getProductsResult).
			Get("/api/v2/products/")
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("response != 200 on getting project '%s'", projectName)
		}
		if getProductsResult.Count == 1 {
			shared.NewLogger("core").Info("project with this name already exists", "projectName", projectName)
			continue
		}

		shared.NewLogger("core").Info("creating new defectdojo project", "projectName", projectName)

		resp, err = client.R().
			SetFormData(map[string]string{
				"name":        projectName,
				"description": fmt.Sprintf("Default description for '%s'", projectName),
				"prod_type":   "1",
				"tags":        projectNameTag,
			}).
			Post("/api/v2/products/")
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("response != 201 on creating project '%s': %s", projectName, resp)
		}
	}

	return nil
}

func uploadResultsToDefectDojo(reposDetails []shared.RepositoryParams) error {
	client := resty.New()
	client.SetHostURL(allUploadResultsOptions.URL)
	client.SetHeader("Authorization", fmt.Sprintf("Token %s", DEFECTDOJO_TOKEN))

	for _, repo := range reposDetails {

		projectName := namespaceToDefectDojoProjectName(repo.Namespace)
		projectNameTag := projectNameToTag(projectName)

		var getProductsResult GetProductsResult
		resp, err := client.R().
			SetQueryParams(map[string]string{
				"tag": projectNameTag,
			}).
			SetResult(&getProductsResult).
			Get("/api/v2/products/")
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("response != 200 on getting project '%s'", projectName)
		}
		if getProductsResult.Count != 1 {
			return fmt.Errorf("found more than one project for '%s'", projectName)
		}

		var engagement DefectDojoEngagement

		currentTime := time.Now()

		resp, err = client.R().
			SetFormData(map[string]string{
				"target_start": currentTime.Format("2006-01-02"),
				"target_end":   currentTime.Format("2006-01-02"),
				"status":       "Completed",
				"product":      strconv.Itoa(getProductsResult.Results[0].ID),
				"name":         repo.RepoName,
			}).
			SetResult(&engagement).
			Post("/api/v2/engagements/")
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("response != 201 on creating engagement %s", resp)
		}

		resultsPath := filepath.Join(shared.GetResultsHome(), allUploadResultsOptions.VCSURL, repo.Namespace, repo.RepoName, "semgrep.raw")

		resp, err = client.R().
			SetFiles(map[string]string{
				"file": resultsPath,
			}).
			SetFormData(map[string]string{
				"engagement": strconv.Itoa(engagement.ID),
				"scan_type":  "SARIF",
				"service":    repo.RepoName,
			}).
			SetResult(&engagement).
			Post("/api/v2/import-scan/")
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("response != 201 on importing scan results %s, %d", resp, resp.StatusCode())
		}
	}

	return nil
}

// uploadResultsCmd represents the uploadResults command
var uploadResultsCmd = &cobra.Command{
	Use:   "upload-results",
	Short: "Upload results to defectdojo",
	// Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("uploadResults called")
		fmt.Printf("DefectDojo URL: %s\n", allUploadResultsOptions.URL)

		reposDetails, err := utils.ReadReposFile2(allUploadResultsOptions.InputFile)
		if err != nil {
			return err
		}

		err = ensureDefectDojoProjectExistance(reposDetails)
		if err != nil {
			return err
		}

		err = uploadResultsToDefectDojo(reposDetails)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadResultsCmd)

	uploadResultsCmd.Flags().StringVar(&allUploadResultsOptions.URL, "url", "http://defectdojo.example.com:8080", "DefectDojo URL")
	uploadResultsCmd.Flags().StringVarP(&allUploadResultsOptions.InputFile, "input", "f", "", "file with list of repos. Results of there repos will be uploaded")
	uploadResultsCmd.Flags().StringVar(&allUploadResultsOptions.ProjectsPrefix, "prefix", "", "projects prefix. Handy for multirepo organizations. Example: 'gitlab.example.com/'")
	uploadResultsCmd.Flags().StringVar(&allUploadResultsOptions.VCSURL, "vcs-url", "", "url to VCS - github.com")
}
