package expose

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	parameterSchema = "schema"
)

var schema string

func NewCommand() *cobra.Command {
	return cmdExpose
}

var cmdExpose = &cobra.Command{
	Use:   "expose ",
	Short: "Expose a json schema an HTTP protocol",
	Run: func(cmd *cobra.Command, args []string) {

		_ = viper.BindPFlag(parameterSchema, cmd.Flags().Lookup(parameterSchema))
		schema = viper.GetString(parameterSchema)
		expose(schema)
	},
}

func expose(filepath string) {

}
