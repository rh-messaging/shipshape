package framework

import (
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/onsi/ginkgo"
)

// Constants available for all test specs related with shipshape framework 
const (
	DeployName = "shipshape"
)

var (
	// Framework instance that holds the generated resources
	Framework *framework.Framework
)

// Create the Framework instance to be used
var _ = ginkgo.BeforeEach(func() {
	// Setup the topology
	Framework = framework.NewFramework(DeployName, framework.TestContext.GetContexts()[0])
}, 60)

// After each test completes, run cleanup actions to save resources (otherwise resources will remain till
// all specs from this suite are done.
var _ = ginkgo.AfterEach(func() {
	Framework.AfterEach()
})

