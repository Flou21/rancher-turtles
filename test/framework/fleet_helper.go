/*
Copyright 2023 SUSE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"bytes"
	"context"
	"os"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FleetCreateGitRepoInput is the input to FleetCreateGitRepo.
type FleetCreateGitRepoInput struct {
	Name             string
	Namespace        string
	Repo             string
	Branch           string
	Paths            []string
	FleetGeneration  int
	ClientSecretName string
	ClusterProxy     framework.ClusterProxy
}

// FleetCreateGitRepo will create and apply a GitRepo resource to the cluster. See the Fleet docs
// for further information: https://fleet.rancher.io/gitrepo-add
func FleetCreateGitRepo(ctx context.Context, input FleetCreateGitRepoInput) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for FleetCreateGitRepo")
	Expect(input.Name).ToNot(BeEmpty(), "Invalid argument. input.Name can't be empty when calling FleetCreateGitRepo")
	Expect(input.Repo).ToNot(BeEmpty(), "Invalid argument. input.Repo can't be empty when calling FleetCreateGitRepo")
	Expect(input.ClusterProxy).ToNot(BeNil(), "Invalid argument. input.Clusterproxy can't be nil when calling FleetCreateGitRepo")

	By("Creating GitRepo from template")

	if input.Namespace == "" {
		input.Namespace = DefaultNamespace
	}

	if input.Branch == "" {
		input.Branch = DefaultBranchName
	}

	t := template.New("fleet-repo-template")
	t, err := t.Parse(gitRepoTemplate)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to pass GitRepo template")

	var renderedTemplate bytes.Buffer
	err = t.Execute(&renderedTemplate, input)
	Expect(err).NotTo(HaveOccurred(), "Failed to execute template")

	By("Applying GitRepo")

	Eventually(func() error {
		return input.ClusterProxy.Apply(ctx, renderedTemplate.Bytes())
	}, retryableOperationTimeout, retryableOperationInterval).Should(Succeed(), "Failed to appl GitRepo")
}

// FleetDeleteGitRepoInput is the input to FleetDeleteGitRepo.
type FleetDeleteGitRepoInput struct {
	Name         string
	Namespace    string
	ClusterProxy framework.ClusterProxy
}

// FleetDeleteGitRepo will delete a GitRepo resource from a cluster.
func FleetDeleteGitRepo(ctx context.Context, input FleetDeleteGitRepoInput) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for FleetDeleteGitRepoInput")
	Expect(input.Name).ToNot(BeEmpty(), "Invalid argument. input.Name can't be empty when calling FleetDeleteGitRepoInput")
	Expect(input.ClusterProxy).ToNot(BeNil(), "Invalid argument. input.Clusterproxy can't be nil when calling FleetDeleteGitRepoInput")

	if input.Namespace == "" {
		input.Namespace = DefaultNamespace
	}

	By("Getting GitRepo from cluster")

	gvkGitRepo := schema.GroupVersionKind{Group: "fleet.cattle.io", Version: "v1alpha1", Kind: "GitRepo"}

	repo := &unstructured.Unstructured{}
	repo.SetGroupVersionKind(gvkGitRepo)
	err := input.ClusterProxy.GetClient().Get(ctx, client.ObjectKey{Namespace: input.Namespace, Name: input.Name}, repo)
	if err != nil {
		if apierrors.IsNotFound(err) {
			By("Skipping deletion as GitRepo not found")

			return
		}
		Expect(err).ShouldNot(HaveOccurred(), "Failed getting GitRepo")
	}

	By("Deleting GitRepo from cluster")

	Eventually(func() error {
		return input.ClusterProxy.GetClient().Delete(ctx, repo)
	}, retryableOperationTimeout, retryableOperationInterval).Should(Succeed(), "Failed to delete GitRepo")
}

// FleetCreateFleetFileInput is the input to FleetCreateFleetFile.
type FleetCreateFleetFileInput struct {
	Namespace string
	FilePath  string
}

// FleetCreateFleetFile will create a fleet.yaml file in the given location.
// See the Fleet docs for further information: https://fleet.rancher.io/ref-fleet-yaml
func FleetCreateFleetFile(ctx context.Context, input FleetCreateFleetFileInput) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for FleetCreateFleetFile")
	Expect(input.FilePath).ToNot(BeEmpty(), "Invalid argument. input.Filepath can't be empty when calling FleetCreateFleetFile")

	if input.Namespace == "" {
		input.Namespace = DefaultNamespace
	}

	t := template.New("fleet-file-template")
	t, err := t.Parse(fleetTemplate)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to pass fleet file template")

	f, err := os.OpenFile(input.FilePath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to create writer for file")

	err = t.Execute(f, input)
	Expect(err).NotTo(HaveOccurred(), "Failed to execute template")
}

const gitRepoTemplate = `
kind: GitRepo
apiVersion: fleet.cattle.io/v1alpha1
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  repo: {{ .Repo }}
  branch: {{ .Branch }}
  forceSyncGeneration: {{ .FleetGeneration }}
  {{- if .ClientSecretName }}
  clientSecretName: {{ .ClientSecretName }}
  {{ end -}}
  paths:
  {{ range .Paths }}
  - {{.}}
  {{ end }}
`

const fleetTemplate = `
namespace: {{ .Namespace }}
defaultNamespace: {{ .Namespace }}
`
