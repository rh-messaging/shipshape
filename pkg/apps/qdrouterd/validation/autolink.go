package validation

import (
	"github.com/interconnectedcloud/qdr-operator/pkg/apis/interconnectedcloud/v1alpha1"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/deployment"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/onsi/gomega"
)

// AutoLinkMapByAddress represents a map whose keys are the
// addresses and the values are maps of AutoLink (Entity) models
// with properties (string) and respective values that can be used
// to compare expected results with an AutoLink entity instance.
type AutoLinkMapByAddress map[string]map[string]interface{}

// ValidateSpecAutoLink asserts that the autoLink models provided through the alMap
// are present across all pods from the given ic instance.
func ValidateSpecAutoLink(ic *v1alpha1.Interconnect, c framework.ContextData, alMap AutoLinkMapByAddress) {
	// Retrieving latest Interconnect
	icNew, err := deployment.GetInterconnect(c, ic.Name)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Iterate through all pods and assert that auto links are available across all instances
	for _, pod := range icNew.Status.PodNames {
		// Same amount of auto links from alMap are expected to be found
		alFound := 0

		// Retrieve autoLinks
		autoLinks, err := qdrmanagement.QdmanageQuery(c, pod, entities.AutoLink{}, nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Loop through returned autoLinks
		for _, e := range autoLinks {
			autoLink := e.(entities.AutoLink)
			alModel, found := alMap[autoLink.Address]
			if !found {
				continue
			}
			// Validating autoLink that exists on alMap
			ValidateEntityValues(autoLink, alModel)
			alFound++
		}

		// Assert that all autoLinks from alMap have been found
		gomega.Expect(alFound).To(gomega.Equal(len(alMap)))
	}
}
