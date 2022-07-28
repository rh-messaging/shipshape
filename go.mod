module github.com/rh-messaging/shipshape

require (
	github.com/artemiscloud/activemq-artemis-operator v1.0.4
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/pkg/errors v0.9.1
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	k8s.io/api v0.24.2
	k8s.io/apiextensions-apiserver v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.70.1 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/controller-runtime v0.12.3
)

go 1.13

//replace github.com/rh-messaging/activemq-artemis-operator => github.com/artemiscloud/activemq-artemis-operator v1.0.4

replace bitbucket.org/ww/goautoneg => github.com/munnerz/goautoneg v0.0.0-20120707110453-a547fc61f48d
