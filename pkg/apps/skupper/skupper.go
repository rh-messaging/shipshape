package skupper

import (
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/framework/operators"
)

type Skupper struct {
	ctx *framework.ContextData
}

func NewSkupper(ctx *framework.ContextData) *Skupper {
	return &Skupper{ctx: ctx}
}

func (s *Skupper) GetOperator() *operators.SkupperOperator {
	operator := s.ctx.OperatorMap[operators.OperatorTypeSkupper]
	return operator.(*operators.SkupperOperator)
}

func (s *Skupper) addGlobalArgs(args []string) []string {
	// Using provided context
	args = append(args, "--namespace", s.ctx.Namespace)
	args = append(args, "--context", s.ctx.Id)
	return args
}
