package command

import (
	"fmt"
	"github.com/ldassonville/json-schema-tools/cmd/jst/command/expose"
	"github.com/ldassonville/json-schema-tools/cmd/jst/command/generate"
	"github.com/ldassonville/json-schema-tools/cmd/jst/command/validate"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "jst",
	Short: "JSON schema tool",
	Long: `JSON schema tool
			JST is a utility tool for generate 
`,
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Println("Hugo Static Site Generator v0.9 -- HEAD")
	},
}

func Execute() {

	rootCmd.AddCommand(validate.NewCommand())
	rootCmd.AddCommand(generate.NewCommand())
	rootCmd.AddCommand(expose.NewCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
