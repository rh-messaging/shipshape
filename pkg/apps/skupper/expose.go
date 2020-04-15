package skupper

import (
	"context"
	"os/exec"
	"strconv"
)

type ExposeFlags struct {
	Address string
	Aggregate string
	Headless bool
	Port int
	TargetPort int
	Protocol string
}

func (s *Skupper) ExposeDeployment(name string, flags ExposeFlags) error {
	return s.runExpose("deployment", name, flags)
}

func (s *Skupper) ExposePods(name string, flags ExposeFlags) error {
	return s.runExpose("pods", name, flags)
}

func (s *Skupper) ExposeStatefulset(name string, flags ExposeFlags) error {
	return s.runExpose("statefulset", name, flags)
}

func (s *Skupper) runExpose(resource string, name string, flags ExposeFlags) error {

	cmdCtx := context.TODO()

	// Building args list
	args := []string{
		"expose",
		resource,
		name,
	}

	// Flags parsing
	if flags.Address != "" {
		args = append(args, "--address", flags.Address)
	}
	if flags.Aggregate != "" {
		args = append(args, "--aggregate", flags.Aggregate)
	}
	if flags.Headless {
		args = append(args, "--headless")
	}
	if flags.Port > 0 {
		args = append(args, "--port", strconv.Itoa(flags.Port))
	}
	if flags.TargetPort > 0 {
		args = append(args, "--target-port", strconv.Itoa(flags.TargetPort))
	}
	if flags.Protocol != "" {
		args = append(args, "--protocol", flags.Protocol)
	}

	// Global args
	args = s.addGlobalArgs(args)

	cmd := exec.CommandContext(cmdCtx, s.GetOperator().SkupperBin(), args...)
	err := cmd.Start()

	return err
}
