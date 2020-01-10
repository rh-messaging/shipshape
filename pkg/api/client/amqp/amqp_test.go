package amqp_test

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

const (
	testTimeout = 10
)

var (
	testPod    v1.Pod
	testClient *amqp.AmqpClientCommon
)

func init() {
	testPod = v1.Pod{
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "PodName"},
	}
	testClient = &amqp.AmqpClientCommon{
		Pod: &testPod,
		Context: framework.ContextData{
			Namespace: "TheNamespace",
			Clients: framework.ClientSet{
				//inserting Mocked clientset, so we can unittest
				//basic functionality without having a cluster.
				KubeClient: fake.NewSimpleClientset(),
			},
		},
		TimedOut:    false,
		Interrupted: false,
	}
}

func updatePodPhase(phase v1.PodPhase, t *testing.T) {
	testPod.Status.Phase = phase
	_, err := testClient.Context.Clients.KubeClient.CoreV1().Pods(testClient.Context.Namespace).Update(testClient.Pod)
	if err != nil {
		t.Fatalf("error updating pod: %v", err)
	}
}

// Testing AmqpClientCommon Status method asserting expected output on every
// possible Pod status phase.
func TestStatus(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	_, err := testClient.Context.Clients.KubeClient.CoreV1().Pods(testClient.Context.Namespace).Create(testClient.Pod)
	if err != nil {
		t.Fatalf("error injecting pod: %v", err)
	}
	testStatus := func(timedOut bool, interrupted bool, e amqp.ClientStatus) {
		testClient.TimedOut = timedOut
		testClient.Interrupted = interrupted
		s := testClient.Status()
		if s != e {
			t.Error("Expected", e,
				"Got", s)
		}
	}

	testStatus(false, true, amqp.Interrupted)
	testStatus(true, false, amqp.Timeout)
	testStatus(true, true, amqp.Timeout)

	podPhaseToAmqpStatus := map[v1.PodPhase]amqp.ClientStatus{
		v1.PodPending:   amqp.Starting,
		v1.PodRunning:   amqp.Running,
		v1.PodSucceeded: amqp.Success,
		v1.PodFailed:    amqp.Error,
		v1.PodUnknown:   amqp.Unknown,
	}

	for testPhase := range podPhaseToAmqpStatus {
		updatePodPhase(testPhase, t)
		testStatus(false, false, podPhaseToAmqpStatus[testPhase])
	}
}

//Testing AmqpClientCommon Running method, again for all possible pod status phases.
func TestRunning(t *testing.T) {
	runningPhases := []v1.PodPhase{v1.PodPending, v1.PodRunning}
	notRunningPhases := []v1.PodPhase{v1.PodSucceeded, v1.PodFailed, v1.PodUnknown}

	testPhases := func(phases []v1.PodPhase, expected bool) {
		var returned bool
		for _, phase := range phases {
			updatePodPhase(phase, t)
			returned = testClient.Running()
			if returned != expected {
				t.Error("Running() method returned", returned, "expected", expected)
			}
		}
	}
	testPhases(runningPhases, true)
	testPhases(notRunningPhases, false)
}
