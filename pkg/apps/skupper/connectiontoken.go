package skupper

import (
	"context"
	"os/exec"
)

type ConnectionTokenFlags struct {
	ClientIdentity string
}

func (s *Skupper) ConnectionToken(outputFile string, flags ConnectionTokenFlags) error {

	cmdCtx := context.TODO()

	// Building args list
	args := []string{
		"connection-token",
		outputFile,
	}

	// Flags parsing
	if flags.ClientIdentity != "" {
		args = append(args, "--client-identity", flags.ClientIdentity)
	}

	// Global args
	args = s.addGlobalArgs(args)

	// Run the command and wait till it finishes (generating the output file)
	cmd := exec.CommandContext(cmdCtx, s.GetOperator().SkupperBin(), args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()

	return err
}
