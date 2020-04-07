package test

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/framework/ginkgowrapper"
	"os"
	"testing"
)

// Initialize once this file is imported, the "init()" method will be called automatically
// by Ginkgo and so, within your test suites you have to explicitly invoke this method
// as it will run your specs and setup the appropriate reporters (if any requested).
// This method MUST be called (otherwise the init() might not be executed).
// The uniqueId is used to help composing the generated JUnit file name (when --report-dir
// is specified when running your tests).
func Initialize(t *testing.T, uniqueId string, description string) {
	framework.HandleFlags()
	gomega.RegisterFailHandler(ginkgowrapper.Fail)
	ginkgo.RunSpecs(t, description)
	os.Setenv("OPERATOR_TESTING", "true")
}

// After suite validation teardown (happens only once per test suite)
var _ = ginkgo.SynchronizedAfterSuite(func() {
	// All nodes tear down
}, func() {
	// Node1 only tear down
	framework.RunCleanupActions(framework.AfterEach)
	framework.RunCleanupActions(framework.AfterSuite)
}, 10)
