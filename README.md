# Shipshape
Kubernetes and OpenShift Operator test framework, written in Go.

# Build status
[![Build Status](https://travis-ci.com/rh-messaging/shipshape.svg?branch=master)](https://travis-ci.com/rh-messaging/shipshape)

# Description

Shipshape is a Kubernetes and OpenShift framework that helps testing all supported
operators (see `pkg/framework/operators/supported.go` for a list of supported operators)
on your running cluster, for end-to-end testing purposes.

The Framework uses Ginkgo and Gomega to help validating that all setup and teardown tasks
have been performed successfully.

# How it works

## Pre-requisites
You must have a running cluster and you must be logged in to your cluster using
an account that has been granted with cluster admin role (because the Framework
will create and remove cluster level resources, like namespaces, roles and CRDs).

The `KUBECONFIG` variable must be set and referring to a kubernetes config that has
the credentials and contexts related with the cluster to be used.

**_Note:_** If no context is specified, the framework will use the current-context
set in your KUBECONFIG file. The framework allows using multiple contexts as well,
and this is a feature that can be used to define tests that run across clusters.
 
## Lifecycle

1. Once a new instance of the Framework is created, a new namespace is created and all
supported operators are deployed to the new namespace(s) (along with all their
dependant resources). It is recommended to create a Framework instance before every
test spec is executed, so they can run  in parallel, independently from each other.

2. After the Framework has been initialized, you can setup your test suite accordingly,
deploying all other resources, as needed. Deployment of application specific resources
should be defined under the "apps" package. See: `pkg/apps` directory for available
helpers.

4. After each test spec completes, it is also recommended to perform a teardown of your
Framework instances, removing all created resources (including the generated namespace).

Your test suite must be defined using Ginkgo (BDD Go test framework). Further info can be 
found at: https://onsi.github.io/ginkgo/.

# Running the end-to-end cluster-tests

Before you can run the end-to-end cluster tests, you have to perform a few steps.
  
## Pre-requisites
* Setup your go environment (install go, set GOPATH, ...)
* Install `kubectl`
* Have a running Kubernetes cluster you can use (or start your own cluster)
* export KUBECONFIG variable
* Log into your cluster or setup your contexts (if not yet done)
* Install ginkgo (see: https://onsi.github.io/ginkgo/)

    ```shell script
    $ go get github.com/onsi/ginkgo/ginkgo
    $ go get github.com/onsi/gomega/...
   ```

## Executing end-to-end tests

Once your cluster is up and running, you can run the test suites by executing:

1. Run all test suites

`make cluster-test`

### End-to-end test overview

At the `test` directory you may find a test suite that demonstrates how to use the Shipshape
framework for writing your tests.

Here is an overview on what is being performed on each file used by the end-to-end test suite:

- test/framework/framework_suite_test.go
  - Your test suite entry point, which must initialize the Shipshape framework and Ginkgo (in this example
    by calling the `Initialize` method)
- test/test_base.go
  - Provides a sample `Initialize` method with mandatory steps for setting up your test suite
- test/framework/setup.go
  - Defines `BeforeEach` and `AfterEach` functions that will be executed by Ginkgo before running
    each test spec (on current suite)
  - A new instance of the Shipshape `Framework` is created in the `BeforeEach` execution
  - The teardown process of the created `Framework` instance is exected in the `AfterEach` execution  
test/framework/framework_test.go
  - Validates that the supported operators have been deployed on the new namespace

# Generating and viewing project UML

- To generat UML you need to install goplantuml (outside of the go.mod path):

```shell script
$ go get github.com/jfeliu007/goplantuml/parser
$ go get github.com/jfeliu007/goplantuml/cmd/goplantuml
$ cd $GOPATH/src/github.com/jfeliu007/goplantuml
$ go install ./...
```

- Once installed just doing `make uml` will generate some `*.puml` files in your current folder matching the go packages.
- To visualize the resulting "PlantUML" files, one option is just to install
  the "PlantUML Visualizer" plugin in your browser (for firefox:  https://addons.mozilla.org/es/firefox/addon/plantuml-visualizer/).
- Then just open the files with your browser or do something like `$ firefox file://${PWD}/framework.puml`
