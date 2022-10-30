// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	clientV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var clientset *kubernetes.Clientset
var config *rest.Config
var apiConfig *clientcmdapi.Config

func KubernetesInit() {

	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	var err error
	apiConfig, err = loader.Load()

	config, err = clientcmd.BuildConfigFromKubeconfigGetter("", loader.Load)

	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}

func KubernetesApiConfig() *clientcmdapi.Config {
	return apiConfig
}

func KubernetesClientset() *kubernetes.Clientset {
	return clientset
}

type VersionResponse struct {
	ClientVersion ClientVersionResponse `json:"clientVersion"`
}

type ClientVersionResponse struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
}

func EnsureVersionOfKubernetesCliSupportsExternalAuth() {
	cmd := exec.Command("kubectl", "version", "--client", "--output=json")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	response := &VersionResponse{}
	json.Unmarshal(out.Bytes(), response)

	major, _ := strconv.Atoi(response.ClientVersion.Major)
	minor, _ := strconv.Atoi(response.ClientVersion.Minor)

	supportsExternalAuth := (major == 1 && minor >= 11) || major >= 2
	if !supportsExternalAuth {
		log.Printf("ERROR: kubectl must be at least version 1.11 to support External Auth.\n")
		log.Printf("Found version was: %d.%d.\n", major, minor)
		log.Printf("To fix this issue, run 'brew install kubernetes-cli' or 'brew upgrade kubernetes-cli'\n")
		log.Fatalf("ABORTING!\n")
	}
}

func EnsureContextExists(newContext string) {
	foundContext := false
	for context := range KubernetesApiConfig().Contexts {
		if context == newContext {
			foundContext = true
		}
	}

	if !foundContext {
		fmt.Printf("%v\n", aurora.Red("Context not found; use one of the list below:"))
		PrintExistingContexts()
		os.Exit(1)
	}

}

func PrintExistingContexts() {
	currentContext := KubernetesApiConfig().CurrentContext
	for context := range KubernetesApiConfig().Contexts {
		if context == currentContext {
			fmt.Printf("* %v\n", aurora.Green(context))
		} else {
			fmt.Printf("  %v\n", context)
		}
	}

}

func EnsureNamespaceExists(namespace string, namespaceList *clientV1.NamespaceList) {
	foundNamespace := false
	for _, ns := range namespaceList.Items {
		if namespace == ns.Name {
			foundNamespace = true
		}
	}

	if !foundNamespace {
		fmt.Printf("%v\n", aurora.Red("Namespace not found; use one of the list below:"))
		PrintExistingNamespaces(namespaceList)
		os.Exit(1)
	}
}

func NamespacesToString(namespaceList *clientV1.NamespaceList) []string {
	namespaces := make([]string, 0, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	return namespaces
}

func PrintExistingNamespaces(namespaceList *clientV1.NamespaceList) {
	currentContext := KubernetesApiConfig().CurrentContext
	context := KubernetesApiConfig().Contexts[currentContext]

	for _, ns := range namespaceList.Items {
		if context.Namespace == ns.Name {
			fmt.Printf("* %v\n", aurora.Green(ns.Name))
		} else {
			fmt.Printf("  %v\n", ns.Name)
		}

	}
}

func SelectPod(promptLabel string) string {
	currentContext := KubernetesApiConfig().CurrentContext
	k8sContextDefinition := KubernetesApiConfig().Contexts[currentContext]

	// query for running pods in current namespace
	podList, _ := KubernetesClientset().CoreV1().Pods(k8sContextDefinition.Namespace).List(context.Background(), metav1.ListOptions{})

	podNames := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if pod.Status.Phase == clientV1.PodRunning {
			podNames = append(podNames, pod.Name)
		}
	}

	if len(podNames) == 1 {
		fmt.Printf("%s found exactly one pod, using this one: %s\n", aurora.Yellow("INFO:"), podNames[0])
		return podNames[0]
	}

	prompt := promptui.Select{
		Label: aurora.Bold(promptLabel),
		Items: podNames,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("%s prompt failed:\n    %v\n", aurora.Red("ERROR:"), err)
		// TODO: get rid of os.Exit here (breaks the outer goroutines)
		os.Exit(1)
	}

	return result
}

func EvalScriptParameter(parameter string) string {
	if strings.HasPrefix(parameter, "eval:") {
		vm := goja.New()
		vm.Set("secret", func(secretName string) map[string]string {
			currentContext := KubernetesApiConfig().CurrentContext
			k8sContextDefinition := KubernetesApiConfig().Contexts[currentContext]
			secret, err := KubernetesClientset().CoreV1().Secrets(k8sContextDefinition.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("%s Secret %s could not be fetched (evaluating %s):\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(secretName), aurora.Bold(parameter), err)
				// TODO: get rid of os.Exit here (breaks the outer goroutines)
				os.Exit(1)
			}

			converted := make(map[string]string, len(secret.Data))
			for k, v := range secret.Data {
				converted[k] = string(v)
			}
			return converted
		})

		vm.Set("configmap", func(configmapName string) map[string]string {
			currentContext := KubernetesApiConfig().CurrentContext
			k8sContextDefinition := KubernetesApiConfig().Contexts[currentContext]
			configMap, err := KubernetesClientset().CoreV1().ConfigMaps(k8sContextDefinition.Namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("%s Config Map %s could not be fetched (evaluating %s):\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(configmapName), aurora.Bold(parameter), err)
				// TODO: get rid of os.Exit here (breaks the outer goroutines)
				os.Exit(1)
			}

			return configMap.Data
		})

		v, err := vm.RunString(parameter[5:])
		if err != nil {
			fmt.Printf("%s Could not evaluate %s:\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(parameter), err)
			// TODO: get rid of os.Exit here (breaks the outer goroutines)
			os.Exit(1)
		}

		converted, isOk := v.Export().(string)
		if !isOk {
			fmt.Printf("%s Could not convert %s:\n    Value: %+v\n", aurora.Red("ERROR:"), aurora.Bold(parameter), v)
			// TODO: get rid of os.Exit here (breaks the outer goroutines)
			os.Exit(1)
		}
		return converted
	} else {
		return parameter
	}

}
