package framework_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"github.com/rh-messaging/shipshape/test/framework"
)

var _ = Describe("Framework", func() {

	It("deploys supported operators on new namespaces", func() {
		By("iterating through contexts and retrieving deployed operators")
		for cn, c := range framework.Framework.ContextMap {
			log.Logf("Context: %s - Unique Name: %s", cn, c.UniqueName)
			for _, o := range c.OperatorMap {
				log.Logf("Operator: %s", o.Name())
				// Validate operator has been deployed properly to given context
				deployment, err := c.GetDeployment(o.Name())
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(deployment).NotTo(gomega.BeNil())

				// Expect operator deployment status is "available"
				gomega.Expect(string(deployment.Status.Conditions[0].Type)).To(gomega.Equal("Available"))
			}
		}
	})

})
