package validation

import (
	"github.com/interconnectedcloud/qdr-operator/pkg/apis/interconnectedcloud/v1alpha1"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"k8s.io/api/core/v1"
)

// ValidateDefaultAddresses verifies that the created addresses match expected ones
func ValidateDefaultAddresses(ic *v1alpha1.Interconnect, c framework.ContextData, pods []v1.Pod) {

	const expectedAddresses = 5

	for _, pod := range pods {
		var defaultAddressesFound = 0

		// Querying addresses on given pod
		addrs, err := qdrmanagement.QdmanageQuery(c, pod.Name, entities.Address{}, nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(len(addrs)).To(gomega.Equal(expectedAddresses))

		// Validates all addresses are present and match expected definition
		for _, entity := range addrs {
			addr := entity.(entities.Address)
			switch addr.Prefix {
			case "closest":
				fallthrough
			case "unicast":
				fallthrough
			case "exclusive":
				ValidateEntityValues(addr, map[string]interface{}{
					"Distribution": entities.DistributionClosest,
				})
				defaultAddressesFound++
			case "multicast":
				fallthrough
			case "broadcast":
				ValidateEntityValues(addr, map[string]interface{}{
					"Distribution": entities.DistributionMulticast,
				})
				defaultAddressesFound++
			}
		}

		// Assert default addresses have been found
		gomega.Expect(expectedAddresses).To(gomega.Equal(defaultAddressesFound))
	}

}
