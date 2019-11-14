package qdrmanagement

import (
	"context"
	"github.com/interconnectedcloud/qdr-operator/pkg/apis/interconnectedcloud/v1alpha1"
	entities2 "github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/rh-messaging/shipshape/pkg/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

// WaitForQdrNodesInPod attempts to retrieve the list of Node Entities
// present on the given pod till the expected amount of nodes are present
// or an error or timeout occurs.
func WaitForQdrNodesInPod(ctxData framework.ContextData, pod v1.Pod, expected int, retryInterval, timeout time.Duration) error {
	// Retry logic to retrieve nodes
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		nodes, err := QdmanageQuery(ctxData, pod.Name, entities2.Node{}, nil)
		if err != nil {
			return false, err
		}
		if len(nodes) != expected {
			return false, nil
		}
		return true, nil
	})
	return err
}

func InterconnectHasExpectedNodes(c framework.ContextData, interconnect *v1alpha1.Interconnect) (bool, error) {
	pods, err := c.ListPodsForDeploymentName(interconnect.Name)
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		nodes, err := QdmanageQuery(c, pod.Name, entities2.Node{}, nil)
		if err != nil {
			return false, err
		}
		if int32(len(nodes)) != interconnect.Spec.DeploymentPlan.Size {
			return false, nil
		}
	}
	return true, nil
}

func InterconnectHasExpectedInterRouterConnections(c framework.ContextData, interconnect *v1alpha1.Interconnect) (bool, error) {
	if interconnect.Spec.DeploymentPlan.Role != v1alpha1.RouterRoleInterior {
		// edge role, nothing to see here
		return true, nil
	}

	pods, err := c.ListPodsForDeploymentName(interconnect.Name)
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		nodes, err := QdmanageQuery(c, pod.Name, entities2.Node{}, nil)
		if err != nil {
			return false, err
		}
		if int32(len(nodes)) != interconnect.Spec.DeploymentPlan.Size-1 {
			return false, nil
		}
	}
	return true, nil
}

// Wait until all the pods belonging to the Interconnect deployment report
// expected node counts, irc's, etc.
func WaitUntilFullInterconnectWithQdrEntities(ctx context.Context, c framework.ContextData, interconnect *v1alpha1.Interconnect) error {

	return framework.RetryWithContext(ctx, framework.RetryInterval, func() (bool, error) {
		// Check that all the qdr pods have the expected node cound
		n, err := InterconnectHasExpectedNodes(c, interconnect)
		if err != nil {
			return false, nil
		}
		if !n {
			return false, nil
		}

		return true, nil
	})
}
