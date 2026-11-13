package cmds

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func promptForConfirmation(prompt string) (bool, error) {
	fmt.Print(prompt + " [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	r, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}
	r = strings.TrimSpace(strings.ToLower(r))

	ok := r == "yes" || r == "y"

	return ok, nil
}
