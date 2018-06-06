package version

import (
	"fmt"

	"github.com/bmeg/arachne/version"
	"github.com/spf13/cobra"
)

// Cmd represents the "version" command
var Cmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}
