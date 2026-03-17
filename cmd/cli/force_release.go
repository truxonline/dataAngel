package cli

import (
	"fmt"
	"strings"
)

type ForceReleaseCommand struct {
	LockID string
}

func ParseForceReleaseCommand(args []string) (*ForceReleaseCommand, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("manque d'arguments")
	}

	for i, arg := range args {
		if arg == "--lock-id" && i+1 < len(args) {
			return &ForceReleaseCommand{LockID: args[i+1]}, nil
		}
		if strings.HasPrefix(arg, "--lock-id=") {
			lockID := strings.TrimPrefix(arg, "--lock-id=")
			return &ForceReleaseCommand{LockID: lockID}, nil
		}
	}

	return nil, fmt.Errorf("option --lock-id manquante")
}
