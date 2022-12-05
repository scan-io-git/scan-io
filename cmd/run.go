/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	utils "github.com/scan-io-git/scan-io/internal/utils"
	// ivcs "github.com/scan-io-git/scan-io/internal/vcs"
	"github.com/scan-io-git/scan-io/pkg/shared"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/utils/pointer"
)

// const IMAGE = "356918957485.dkr.ecr.eu-west-2.amazonaws.com/i-am-first"
const IMAGE = "scanio"
const AWS_DEFAULT_REGION = "eu-west-2"
const S3_BUCKET = "my-s3-bucket-q97843yt9"
const DEFAULT_JOB_HELM_CHART_PATH = "helm/scanio-helm/scanio-job"

type RunOptions struct {
	VCSPlugin     string
	VCSURL        string
	Repos         []string
	RmExts        []string
	ScannerPlugin string
	StorageType   string
	S3Bucket      string
	Runtime       string
	Image         string
	// Experiment    string
	InputFile string
	Jobs      int
}

var o RunOptions

func getRepoID(repo string) string {
	return filepath.Join(o.VCSURL, repo)
}

func getS3Path(repo string) string {
	return filepath.Join(getRepoID(repo), fmt.Sprintf("%s.raw", o.ScannerPlugin))
}

func getResultsFolder(repo string) string {
	return filepath.Join(shared.GetResultsHome(), getRepoID(repo))
}

func getResultsPath(repo string) string {
	return filepath.Join(shared.GetResultsHome(), getS3Path(repo))
}

func fetch(repo string) {
	logger := shared.NewLogger("core")
	logger.Info("Fetching starting", "VCSURL", o.VCSURL, "repo", repo)

	targetFolder := shared.GetRepoPath(o.VCSURL, repo)

	shared.WithPlugin("plugin-vcs", shared.PluginTypeVCS, o.VCSPlugin, func(raw interface{}) {

		vcs := raw.(shared.VCS)
		args := shared.VCSFetchRequest{
			CloneURL: repo,
			//VCSURL:       o.VCSURL,
			TargetFolder: targetFolder,
		}
		err := vcs.Fetch(args)
		if err != nil {
			logger.Debug("Fetch error", "err", err)
		} else {
			logger.Debug("Removing files with some extentions", "extentions", o.RmExts)
			findByExtAndRemove(targetFolder, o.RmExts)
		}
	})

	logger.Info("All fetch operations are finished.")
}

func scan(repo string) {
	logger := shared.NewLogger("core")
	logger.Info("Scan starting", "scanner", o.ScannerPlugin, "VCSURL", o.VCSURL, "repo", repo)

	repoPath := shared.GetRepoPath(o.VCSURL, repo)

	err := os.MkdirAll(getResultsFolder(repo), 0666)
	if err != nil {
		// logger.Warn("error creating results folder", "err", err)
		panic(err)
	}

	shared.WithPlugin("plugin-scanner", shared.PluginTypeScanner, o.ScannerPlugin, func(raw interface{}) {
		raw.(shared.Scanner).Scan(shared.ScannerScanRequest{
			RepoPath:    repoPath,
			ResultsPath: getResultsPath(repo),
		})
	})

	logger.Debug("Scan finished.")
}

func uploadResults(repo string) {
	logger := shared.NewLogger("core")
	logger.Info("Uploading results", "resultsPath", getResultsPath(repo), "storage-type", o.StorageType, "bucket", o.S3Bucket, "path", getS3Path(repo))

	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(AWS_DEFAULT_REGION),
	}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	path := getResultsPath(repo)
	// if o.Experiment == "upload" {
	// 	path = filepath.Join(os.Getenv("HOME"), ".bashrc")
	// }
	f, err := os.Open(path)
	if err != nil {
		// panic(fmt.Errorf("failed to open file %q, %v", f, err))
		logger.Warn("failed to open file with results", "file", f, "err", err)
	}
	defer f.Close()

	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(o.S3Bucket),
		Key:    aws.String(getS3Path(repo)),
		Body:   f,
	})
	if err != nil {
		// panic(fmt.Errorf("failed to upload file, %v", err))
		logger.Warn("failed to upload results file", "err", err)
	}
	// fmt.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))
	logger.Info("uploaded results", "bucket", o.S3Bucket, "path", getS3Path(repo), "result", result)
}

func getPodSpec(jobID string, repo string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("job-%s", jobID),
		},
		Spec: batchv1.JobSpec{
			Parallelism:             pointer.Int32(1),
			Completions:             pointer.Int32(1),
			BackoffLimit:            pointer.Int32(0),
			TTLSecondsAfterFinished: pointer.Int32(3600),
			// Selector: &metav1.LabelSelector{
			// 	MatchLabels: map[string]string{
			// 		"app": "demo",
			// 	},
			// },
			Template: apiv1.PodTemplateSpec{
				// ObjectMeta: metav1.ObjectMeta{
				// 	Labels: map[string]string{
				// 		"app": "demo",
				// 	},
				// },
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  fmt.Sprintf("container-%s", jobID),
							Image: o.Image,
							// Command: []string{"bash", "-c", "echo $AWS_SECRET_ACCESS_KEY | sha1sum"},
							Command: []string{
								"scanio", "run",
								"--vcs-plugin", o.VCSPlugin,
								"--vcs-url", o.VCSURL,
								"--scanner-plugin", o.ScannerPlugin,
								"--repos", repo,
								"--storage-type", "s3",
								"--s3bucket", o.S3Bucket,
							},
							Env: []apiv1.EnvVar{
								{
									Name: "AWS_ACCESS_KEY_ID",
									ValueFrom: &apiv1.EnvVarSource{
										SecretKeyRef: &apiv1.SecretKeySelector{
											LocalObjectReference: apiv1.LocalObjectReference{Name: "s3"},
											Key:                  "aws_access_key_id",
											Optional:             pointer.Bool(false),
										},
									},
								},
								{
									Name: "AWS_SECRET_ACCESS_KEY",
									ValueFrom: &apiv1.EnvVarSource{
										SecretKeyRef: &apiv1.SecretKeySelector{
											LocalObjectReference: apiv1.LocalObjectReference{Name: "s3"},
											Key:                  "aws_secret_access_key",
											Optional:             pointer.Bool(false),
										},
									},
								},
							},
						},
					},
					RestartPolicy: apiv1.RestartPolicyNever,
					// ServiceAccountName: "my-service-account",
				},
			},
		},
	}
}

func tryGetFromClusterConfig() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
		// panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
		// panic(err)
	}
	return clientset, nil
}

func tryGetFromKubeConfig() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
		// panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
		// panic(err)
	}
	return clientset, nil
}

func getNewJobsClient() v1.JobInterface {

	clientset, err1 := tryGetFromClusterConfig()
	if err1 == nil {
		return clientset.BatchV1().Jobs(apiv1.NamespaceDefault)
	}

	clientset, err2 := tryGetFromKubeConfig()
	if err2 == nil {
		return clientset.BatchV1().Jobs(apiv1.NamespaceDefault)
	}

	panic(fmt.Errorf("cant create kubernetes client. For cluster config: %w. For kube config: %w", err1, err2))
}

func fetchResults(repo string) {
	logger := shared.NewLogger("core")
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: "s3",
		Config: aws.Config{
			Region: aws.String(AWS_DEFAULT_REGION),
		},
	})
	if err != nil {
		logger.Warn("unable to create s3 client to fetch scan results", "err", err)
		return
	}

	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(o.S3Bucket),
		Key:    aws.String(getS3Path(repo)),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				logger.Warn("s3 error. ErrCodeNoSuchKey", "aerr", aerr)
				// fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
			case s3.ErrCodeInvalidObjectState:
				logger.Warn("s3 error. ErrCodeInvalidObjectState", "aerr", aerr)
				// fmt.Println(s3.ErrCodeInvalidObjectState, aerr.Error())
			default:
				logger.Warn("s3 error", "aerr", aerr)
				// fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			logger.Warn("s3 error", "err", err)
			// fmt.Println(err.Error())
		}
		return
	}
	defer result.Body.Close()

	// fmt.Println(result)
	err = os.MkdirAll(getResultsFolder(repo), 0777)
	if err != nil {
		logger.Warn("error creating results folder", "err", err)
		return
	}

	f, err := os.Create(getResultsPath(repo))
	if err != nil {
		logger.Warn("failed to create file with results", "file", f, "err", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, result.Body)
	if err != nil {
		logger.Warn("failed to write results to a file", "file", f, "result", result, "err", err)
		return
	}
}

func runWithHelm(repos []string) {
	logger := shared.NewLogger("core")
	logger.Info("runWithHelm")

	values := make([]interface{}, len(repos))
	for i := range repos {
		values[i] = repos[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(o.Jobs, values, func(i int, value interface{}) {
		repo := value.(string)
		logger.Info("runWithHelm Goroutine started", "#", i+1, "repo", repo, "jobs", o.Jobs)

		jobID := uuid.New()
		logger.Debug("runWithHelm jobID", "jobID", jobID)

		remoteCommandArgs := []string{
			"scanio", "run",
			"--vcs-plugin", o.VCSPlugin,
			"--vcs-url", o.VCSURL,
			"--scanner-plugin", o.ScannerPlugin,
			"--repos", repo,
		}

		if o.StorageType == "remote" {
			remoteCommandArgs = append(remoteCommandArgs, "--storage-type", "local")
		} else if o.StorageType == "local" || o.StorageType == "s3" {
			remoteCommandArgs = append(remoteCommandArgs, "--storage-type", "s3", "--s3bucket", o.S3Bucket)
		}
		logger.Debug("runWithHelm jobID", "command", remoteCommandArgs)
		jobCommand := fmt.Sprintf("command={%s}", strings.Join(remoteCommandArgs, ","))

		jobChartPath := DEFAULT_JOB_HELM_CHART_PATH
		if path := os.Getenv("JOB_HELM_CHART_PATH"); path != "" {
			jobChartPath = path
		}

		cmd := exec.Command("helm", "install", jobID.String(), jobChartPath,
			"--set", jobCommand,
			"--set", fmt.Sprintf("image.repository=%s", IMAGE),
			"--set", "image.tag=latest",
			"--set", fmt.Sprintf("suffix=%s", jobID.String()),
		)
		logger.Debug("runWithHelm jobID", "cmd", cmd)
		if err := cmd.Run(); err != nil {
			logger.Debug("helm install error", "err", err)
			log.Fatal(err)
		}

		jobsClient := getNewJobsClient()

		jobName := fmt.Sprintf("scanio-job-%s", jobID.String())

		logger.Info("Waiting the job", "jobName", jobName)
		for {
			job, err := jobsClient.Get(context.Background(), jobName, metav1.GetOptions{})
			if err != nil {
				panic(err)
			}
			if job.Status.Succeeded > 0 || job.Status.Failed == *job.Spec.BackoffLimit+1 {
				break
			}
		}

		if o.StorageType == "local" {
			logger.Info("Fetching results", "#", i+1, "jobID", jobID)
			fetchResults(repo)
		}

		cmd = exec.Command("helm", "uninstall", jobID.String())
		if err := cmd.Run(); err != nil {
			logger.Debug("helm uninstall error", "err", err)
			log.Fatal(err)
		}
	})
}

func runInK8S(repos []string) {
	logger := shared.NewLogger("core")

	values := make([]interface{}, len(repos))
	for i := range repos {
		values[i] = repos[i]
	}

	shared.ForEveryStringWithBoundedGoroutines(o.Jobs, values, func(i int, value interface{}) {
		repo := value.(string)
		logger.Info("Goroutine started", "#", i+1, "repo", repo)

		jobsClient := getNewJobsClient()

		jobID := uuid.New()
		jobName := fmt.Sprintf("job-%s", jobID.String())
		myJob := getPodSpec(jobID.String(), repo)

		logger.Info("Running k8s job", "jobID", jobID)
		_, err := jobsClient.Create(context.Background(), myJob, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}

		logger.Info("Waiting the job", "jobName", jobName)
		for {
			job, err := jobsClient.Get(context.Background(), jobName, metav1.GetOptions{})
			if err != nil {
				panic(err)
			}
			if job.Status.Succeeded > 0 || job.Status.Failed == *job.Spec.BackoffLimit+1 {
				break
			}
		}

		if o.StorageType == "local" {
			logger.Info("Fetching results", "jobID", jobID)
			fetchResults(repo)
		}
	})
}

func getReposToProcess() []string {
	repos := []string{}
	for _, r := range o.Repos {
		repos = append(repos, r)
	}

	if len(o.InputFile) > 0 {
		reposFromFile, err := utils.ReadReposFile(o.InputFile)
		if err != nil {
			log.Fatal(err)
		}
		for _, r := range reposFromFile {
			repos = append(repos, r)
		}
	}

	return repos
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	// 	Long: `A longer description that spans multiple lines and likely contains examples
	// and usage of using your command. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
		cmd.Flags().Parse(args)
		repos := getReposToProcess()
		shared.NewLogger("core").Info("Run", "vcsPlugin", o.VCSPlugin, "VCSURL", o.VCSURL, "Runtime", o.Runtime)
		// if o.Experiment == "upload" {
		// 	for _, repo := range repos {
		// 		uploadResults(repo)
		// 	}
		// 	return
		// }
		if o.Runtime == "local" {
			for _, repo := range repos {
				fetch(repo)
				scan(repo)
				if o.StorageType == "s3" {
					uploadResults(repo)
				}
			}
		}
		if o.Runtime == "k8s" {
			runInK8S(repos)
		}
		if o.Runtime == "helm" {
			runWithHelm(repos)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	runCmd.Flags().StringVar(&o.VCSPlugin, "vcs-plugin", "github", "vcs plugin name")
	runCmd.Flags().StringVar(&o.VCSURL, "vcs-url", "github.com", "vcs base url")
	runCmd.Flags().StringSliceVar(&o.Repos, "repos", []string{}, "repo path to scan")
	runCmd.Flags().StringVarP(&o.InputFile, "input", "f", "", "repo path to scan")
	runCmd.Flags().StringSliceVar(&o.RmExts, "rm-ext", strings.Split("csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe", ","), "Files with extention to remove automatically after checkout")
	runCmd.Flags().StringVar(&o.ScannerPlugin, "scanner-plugin", "semgrep", "scanner plugin name")
	runCmd.Flags().StringVar(&o.StorageType, "storage-type", "local", "storage type")
	runCmd.Flags().StringVar(&o.S3Bucket, "s3bucket", S3_BUCKET, "s3 bucket name when storage-type 's3' in use")
	// runCmd.Flags().StringVar(&o.S3Path, "s3path", "", "s3 path when storage-type 's3' in use")
	runCmd.Flags().StringVar(&o.Runtime, "runtime", "local", "runtime 'local' or 'k8s'")
	runCmd.Flags().StringVar(&o.Image, "image", IMAGE, "container image to scan in k8s")
	runCmd.Flags().IntVarP(&o.Jobs, "jobs", "j", 1, "k8s jobs to run in parallel")

	// runCmd.Flags().StringVar(&o.Experiment, "experiment", "", "")
}
