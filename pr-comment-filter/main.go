package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tkn "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tknclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ORG_NAME = "giantswarm"
)

var (
	env map[string]string

	// Examples:
	// /run build-and-publish
	// /run test-cluster-create PRIVATE_NETWORK=true
	// /run test-cluster-create PREVIOUS_VERSION=1.2.6
	// /run test-cluster-upgrade PRIVATE_NETWORK=false PREVIOUS_VERSION=1.2.6
	triggerFormat = regexp.MustCompile(`(?mi)^\/run (?P<pipeline>\S+) ?(?P<args>(?:[A-Z_]+=\S+ ?)*)(\r|\n|$)`)

	tektonClient *tknclient.Clientset
	kubeClient   kubernetes.Interface
)

type Trigger struct {
	FullTrigger  string
	PipelineName string
	Args         map[string]string
}

func init() {
	if os.Getenv("COMMENT") == "" {
		fmt.Println("No comment provided")
		os.Exit(0)
	}

	env = map[string]string{
		"URL":              os.Getenv("URL"),
		"NUMBER":           os.Getenv("NUMBER"),
		"TITLE":            os.Getenv("TITLE"),
		"BODY":             os.Getenv("BODY"),
		"GIT_REVISION":     os.Getenv("GIT_REVISION"),
		"CLONE_URL":        os.Getenv("CLONE_URL"),
		"REPO_NAME":        os.Getenv("REPO_NAME"),
		"REPO_ORG":         os.Getenv("REPO_ORG"),
		"CHANGED_FILES":    os.Getenv("CHANGED_FILES"),
		"COMMENT":          os.Getenv("COMMENT"),
		"PREVIOUS_COMMENT": os.Getenv("PREVIOUS_COMMENT"),
		"COMMENT_ID":       os.Getenv("COMMENT_ID"),
		"COMMENT_URL":      os.Getenv("COMMENT_URL"),
		"USER_LOGIN":       os.Getenv("USER_LOGIN"),
		"USER_TYPE":        os.Getenv("USER_TYPE"),
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	tektonClient, err = tknclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Printf("Filtering PR comments for valid triggers. Repo = %s, PR = %s\n", env["REPO_NAME"], env["NUMBER"])

	ctx := context.Background()

	if !isUserAllowed(ctx, env["USER_LOGIN"], env["USER_TYPE"]) {
		fmt.Printf("User not permitted to trigger pipelines. User: %s, Type: %s\n", env["USER_LOGIN"], env["USER_TYPE"])
		return
	}

	triggerMatches := triggerFormat.FindAllStringSubmatch(os.Getenv("COMMENT"), -1)

	// For comments on PRs we don't get all the details of the PR so may need to fetch those from the API
	if len(triggerMatches) > 0 && env["GIT_REVISION"] == "" {
		oClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
		))
		ghClient := github.NewClient(oClient)
		prNumber, err := strconv.Atoi(env["NUMBER"])
		if err != nil {
			panic("Failed to parse PR number to int")
		}
		pr, _, err := ghClient.PullRequests.Get(ctx, env["REPO_ORG"], env["REPO_NAME"], prNumber)
		if err != nil {
			fmt.Println("Failed to get PR details from GitHub API", err)
			os.Exit(1)
		}
		env["GIT_REVISION"] = *pr.Head.SHA
	}

	for _, match := range triggerMatches {
		trigger := parseTriggerLine(match)

		namespace := "tekton-pipelines"
		if val, ok := trigger.Args["NAMESPACE"]; ok && val != "" {
			namespace = trigger.Args["NAMESPACE"]
		}

		// Check if we can find the pipeline in the cluster
		pipeline, err := getPipeline(ctx, trigger.PipelineName, namespace)
		if err != nil {
			fmt.Printf("Failed to find pipeline '%s', skipping\n", trigger.PipelineName)
			continue
		}
		namespace = pipeline.ObjectMeta.Namespace

		// Check if we can find an appropriately named ServiceAccount or fallback to using `default`
		serviceAccountName := trigger.PipelineName
		serviceAccount, err := getServiceAccount(ctx, serviceAccountName, namespace)
		if err != nil {
			fmt.Printf("Failed to find ServiceAccount, skipping\n")
			continue
		}
		serviceAccountName = serviceAccount.ObjectMeta.Name

		// Support defining a pipeline timeout as an annotation on the Pipeline resource
		pipelineTimeout, err := time.ParseDuration(getAnnotationOrDefault(pipeline.ObjectMeta.Annotations, "tekton.dev/pipeline-timeout", "1h"))
		if err != nil {
			pipelineTimeout, _ = time.ParseDuration("1h")
		}

		// Support defining the storage class for the pipeline workspace
		workspaceStorageClass := getAnnotationOrDefault(pipeline.ObjectMeta.Annotations, "cicd.giantswarm.io/storage-class", "efs-sc")
		workspaceStorageClassAccessMode := corev1.ReadWriteOnce
		if workspaceStorageClass == "efs-sc" {
			workspaceStorageClassAccessMode = corev1.ReadWriteMany
		}

		pipelineRun := &tkn.PipelineRun{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: fmt.Sprintf("pr-%s-%s-%s", env["REPO_NAME"], env["NUMBER"], trigger.PipelineName),
				Namespace:    namespace,
				Labels: map[string]string{
					"cicd.giantswarm.io/repo":     env["REPO_NAME"],
					"cicd.giantswarm.io/pr":       env["NUMBER"],
					"cicd.giantswarm.io/revision": env["GIT_REVISION"],
				},
				Annotations: map[string]string{
					"cicd.giantswarm.io/url": env["URL"],
				},
			},
			Spec: tkn.PipelineRunSpec{
				PipelineRef: &tkn.PipelineRef{
					Name: trigger.PipelineName,
				},
				Timeouts: &tkn.TimeoutFields{
					Pipeline: &v1.Duration{Duration: pipelineTimeout},
				},
				Params: []tkn.Param{},
				TaskRunTemplate: tkn.PipelineTaskRunTemplate{
					ServiceAccountName: serviceAccountName,
					PodTemplate: &pod.Template{
						ImagePullSecrets: []corev1.LocalObjectReference{
							{
								Name: "quay-imagepull-secret",
							},
						},
					},
				},
				Workspaces: []tkn.WorkspaceBinding{
					{
						Name: "shared",
						VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
							Spec: corev1.PersistentVolumeClaimSpec{
								StorageClassName: &workspaceStorageClass,
								AccessModes: []corev1.PersistentVolumeAccessMode{
									workspaceStorageClassAccessMode,
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("5Gi"),
									},
								},
							},
						},
					},
				},
			},
		}

		// Populate params with PR details
		for key, val := range env {
			pipelineRun.Spec.Params = append(pipelineRun.Spec.Params, tkn.Param{
				Name: key,
				Value: tkn.ParamValue{
					Type:      tkn.ParamTypeString,
					StringVal: val,
				},
			})
		}

		// Populate params with trigger args
		for key, val := range trigger.Args {
			pipelineRun.Spec.Params = append(pipelineRun.Spec.Params, tkn.Param{
				Name: key,
				Value: tkn.ParamValue{
					Type:      tkn.ParamTypeString,
					StringVal: val,
				},
			})
		}

		fmt.Printf("Creating new PipelineRun - %s\n", trigger.PipelineName)

		_, err = tektonClient.TektonV1().PipelineRuns(namespace).Create(ctx, pipelineRun, v1.CreateOptions{})
		if err != nil {
			fmt.Println("Failed to create new PipelineRun: ", err)
		}
	}

	if len(triggerMatches) == 0 {
		fmt.Println("No triggers found, nothing to do")
	} else {
		fmt.Println("All triggers processed")
	}
}

func parseTriggerLine(triggerLine []string) Trigger {
	trigger := Trigger{
		FullTrigger:  triggerLine[0],
		PipelineName: triggerLine[1],
		Args:         map[string]string{},
	}

	args := strings.TrimSpace(triggerLine[2])
	for _, arg := range strings.Split(args, " ") {
		if arg != "" {
			parts := strings.SplitN(arg, "=", 2)
			trigger.Args[parts[0]] = parts[1]
		}
	}

	return trigger
}

func getPipeline(ctx context.Context, pipelineName string, namespace string) (*tkn.Pipeline, error) {
	pipeline, err := tektonClient.TektonV1().Pipelines(namespace).Get(ctx, pipelineName, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return pipeline, err
	} else if errors.IsNotFound(err) && namespace != "tekton-pipelines" {
		pipeline, err = getPipeline(ctx, pipelineName, "tekton-pipelines")
	}

	return pipeline, err
}

func getServiceAccount(ctx context.Context, serviceAccountName string, namespace string) (*corev1.ServiceAccount, error) {
	serviceAccount, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(ctx, serviceAccountName, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return serviceAccount, err
	} else if errors.IsNotFound(err) && serviceAccountName != "default" {
		serviceAccount, err = getServiceAccount(ctx, "default", namespace)
	}

	return serviceAccount, err
}

func stringToPtr(s string) *string {
	return &s
}

func isUserAllowed(ctx context.Context, userLogin, userType string) bool {
	if strings.ToLower(userType) == "user" && userLogin != "" {
		oClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
		))
		ghClient := github.NewClient(oClient)

		membership, _, err := ghClient.Organizations.GetOrgMembership(ctx, userLogin, ORG_NAME)
		if err != nil {
			fmt.Println("Failed to get org membership from GitHub: ", err)
			return false
		}

		return *membership.State == "active"
	}

	return false
}

func getAnnotationOrDefault(annotations map[string]string, targetKey string, defaultValue string) string {
	val, ok := annotations[targetKey]
	if ok {
		return val
	}
	return defaultValue
}
