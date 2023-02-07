package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	tkn "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tknclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	comment string
	env     map[string]string

	// Examples:
	// /run build-and-publish
	// /run test-cluster-create PRIVATE_NETWORK=true
	// /run test-cluster-create PREVIOUS_VERSION=1.2.6
	// /run test-cluster-upgrade PRIVATE_NETWORK=false PREVIOUS_VERSION=1.2.6
	triggerFormat = regexp.MustCompile(`(?mi)^\/run (?P<pipeline>\S+) ?(?P<args>(?:[A-Z_]+=\S+ ?)*)[\n|$]`)

	tektonClient *tknclient.Clientset
	kubeClient   kubernetes.Interface
)

type Trigger struct {
	FullTrigger  string
	PipelineName string
	Args         map[string]string
}

func init() {
	comment := os.Getenv("COMMENT")
	if comment == "" {
		panic("The COMMENT environment variable is required")
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
		"MERGEABLE_STATE":  os.Getenv("MERGEABLE_STATE"),
		"COMMENT":          os.Getenv("COMMENT"),
		"PREVIOUS_COMMENT": os.Getenv("PREVIOUS_COMMENT"),
		"COMMENT_ID":       os.Getenv("COMMENT_ID"),
		"COMMENT_URL":      os.Getenv("COMMENT_URL"),
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	tektonClient, err = tknclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	kubeClient = kubeclient.Get(context.Background())
}

func main() {
	fmt.Printf("Filtering PR comments for valid triggers. Repo = %s, PR = %s\n", env["REPO_NAME"], env["NUMBER"])

	ctx := context.Background()

	triggerMatches := triggerFormat.FindAllStringSubmatch(comment, -1)
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
				Params: []tkn.Param{},
				TaskRunTemplate: tkn.PipelineTaskRunTemplate{
					ServiceAccountName: serviceAccountName,
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
