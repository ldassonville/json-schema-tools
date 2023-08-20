package generate

import (
	"github.com/ldassonville/json-schema-tools/internal/markdown"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return cmdGenerate
}

var cmdGenerate = &cobra.Command{
	Use:   "generate",
	Short: "Generate the markdown",
	Run: func(cmd *cobra.Command, args []string) {

		markdown.GenerateMarkdown("testcases/values-definition.json", "schema.md")
	},
}
