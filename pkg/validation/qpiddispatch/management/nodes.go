package management

import (
	"github.com/fgiorgetti/shipshape/pkg/framework"
	"github.com/fgiorgetti/shipshape/pkg/framework/qdrmanagement"
	"github.com/onsi/gomega"
)

// ValidateRoutersInNetwork uses qdmanage query to retrieve nodes in the router network.
// It iterates through all pods available in the provided context and deployment and waits
// till the expected amount of nodes are present or till it times out.
func ValidateRoutersInNetwork(ctx *framework.ContextData, deploymentName string, expectedCount int) {
	// Retrieves pods
	podList, err := ctx.ListPodsForDeploymentName(deploymentName)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(podList.Items).To(gomega.HaveLen(1))

	// Iterate through pods and execute qdmanage query across all pods
	for _, pod := range podList.Items {
		// Wait till expected amount of nodes are present or till it times out
		err := qdrmanagement.WaitForQdrNodesInPod(*ctx, pod, expectedCount, framework.RetryInterval, framework.Timeout)
		gomega.Expect(err).To(gomega.BeNil())
	}
}
