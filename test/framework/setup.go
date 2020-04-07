package framework

import (
	"github.com/onsi/ginkgo"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"os"
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
	// Set environment variable to help operators behave differently when running shipshape tests
	os.Setenv("OPERATOR_TESTING", "true")

	// Setup the topology
	builder := framework.NewFrameworkBuilder(DeployName)
	Framework = builder.Build()
}, 60)

// After each test completes, run cleanup actions to save resources (otherwise resources will remain till
// all specs from this suite are done.
var _ = ginkgo.AfterEach(func() {
	Framework.AfterEach()
})

