package skupper

import (
	"context"
	"os/exec"
	"strconv"
)

type ConnectFlags struct {
	ConnectionName string
	Cost int
}

func (s *Skupper) Connect(tokenFile string, flags ConnectFlags) error {

	cmdCtx := context.TODO()

	// Building args list
	args := []string{
		"connect",
		tokenFile,
	}

	// Flags parsing
	if flags.ConnectionName != "" {
		args = append(args, "--connection-name", flags.ConnectionName)
	}
	if flags.Cost > 0 {
		args = append(args, "--cost", strconv.Itoa(flags.Cost))
	}

	// Global args
	args = s.addGlobalArgs(args)

	cmd := exec.CommandContext(cmdCtx, s.GetOperator().SkupperBin(), args...)
	err := cmd.Start()

	return err
}
