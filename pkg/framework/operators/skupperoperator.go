package operators

import (
	"context"
	"os"
	"os/exec"

	appsv1 "k8s.io/api/apps/v1"
)

// SkupperOperatorBuilder helps building a skupper operator that uses
//                        the CLI skupper. Once the API is available
//                        we can create a second skupper operator to
//                        test both api and cli.
type SkupperOperatorBuilder struct {
	BaseOperatorBuilder
	skupper SkupperOperator
}

func (b *SkupperOperatorBuilder) SkupperId(id string) {
	b.skupper.Id = id
}

func (b *SkupperOperatorBuilder) ClusterLocal(local bool) {
	b.skupper.ClusterLocal = local
}

func (b *SkupperOperatorBuilder) WithSkupperPath(path string) {
	b.skupper.SkupperPath = path
}

func (s *SkupperOperatorBuilder) OperatorType() OperatorType {
	return OperatorTypeSkupper
}

//
// Future use
//

func (b *SkupperOperatorBuilder) WithQdrouterdImage(image string) {
	b.skupper.QdrouterdImage = image
}

func (b *SkupperOperatorBuilder) WithControllerImage(image string) {
	b.skupper.ControllerImage = image
}

func (b *SkupperOperatorBuilder) WithProxyImage(image string) {
	b.skupper.ProxyImage = image
}

//
// END OF - Future use
//

func (b *SkupperOperatorBuilder) Build() (OperatorSetup, error) {
	if err := b.skupper.InitFromBaseOperatorBuilder(&b.BaseOperatorBuilder); err != nil {
		return &b.skupper, err
	}
	b.skupper.restConfig = b.restConfig
	b.skupper.rawConfig = b.rawConfig
	// This is set when running shipshape cluster test
	if os.Getenv("OPERATOR_TESTING") != "" {
		b.ClusterLocal(true)
	}
	return &b.skupper, nil
}

type SkupperOperator struct {
	BaseOperator
	ClusterLocal    bool
	Id              string
	QdrouterdImage  string
	ControllerImage string
	ProxyImage      string
	SkupperPath     string
}

func (s *SkupperOperator) SkupperBin() string {
	return s.SkupperPath + "skupper"
}

func (s *SkupperOperator) Name() string {
	return s.operatorName
}

func (s *SkupperOperator) Setup() error {
	// run skupper init with provided flags from builder
	cmdCtx := context.TODO()

	// Building args list
	args := []string{
		"init",
	}
	if s.ClusterLocal {
		args = append(args, "--cluster-local")
	}

	// Using provided context
	args = append(args, "--namespace", s.namespace)
	args = append(args, "--context", s.rawConfig.CurrentContext)

	cmd := exec.CommandContext(cmdCtx, s.SkupperBin(), args...)
	err := cmd.Start()

	return err
}

func (s *SkupperOperator) TeardownEach() error {
	return nil
}

func (s *SkupperOperator) TeardownSuite() error {
	// run skupper delete
	cmdCtx := context.TODO()

	// Building args list
	args := []string{
		"delete",
	}

	// Using provided context
	args = append(args, "--namespace", s.namespace)
	args = append(args, "--context", s.rawConfig.CurrentContext)

	cmd := exec.CommandContext(cmdCtx, s.SkupperBin(), args...)
	err := cmd.Start()

	return err
}

func (b *SkupperOperator) UpdateDeployment(deployment *appsv1.Deployment) error {
	return b.BaseOperator.UpdateDeployment(deployment)
}

func (b *SkupperOperator) DeleteDeployment() error {
	return b.BaseOperator.DeleteDeployment()
}

func (b *SkupperOperator) CreateDeployment() error {
	return b.BaseOperator.CreateDeployment()
}

func (b *SkupperOperator) GetDeployment() (*appsv1.Deployment, error) {
	return b.BaseOperator.GetDeployment()
}
