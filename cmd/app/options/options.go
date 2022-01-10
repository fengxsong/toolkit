package options

import (
	"os"

	"github.com/spf13/cobra"
)

func GetEnvWithDefault(key string, defVal string) string {
	if val := os.Getenv(key); len(val) > 0 {
		return val
	}
	return defVal
}

func ExecuteRootPersistentPreRunE(cmd *cobra.Command, args []string) error {
	if root := cmd.Root(); root != nil {
		if root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(cmd, args); err != nil {
				return err
			}
		}
	}
	return nil
}
