package validate

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	parameterSchema = "schema"

	parameterData = "data"

	parameterBase = "base"
)

var schema string

var basepath string

var data []string

func NewCommand() *cobra.Command {

	var cmdGenerate = &cobra.Command{
		Use:   "validate",
		Short: "Validate the schema",
		//Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			_ = viper.BindPFlag(parameterSchema, cmd.Flags().Lookup(parameterSchema))
			schema = viper.GetString(parameterSchema)

			_ = viper.BindPFlag(parameterData, cmd.Flags().Lookup(parameterData))
			data = viper.GetStringSlice(parameterData)

			_ = viper.BindPFlag(parameterBase, cmd.Flags().Lookup(parameterBase))
			basepath = viper.GetString(parameterBase)

			validateJsons(basepath, schema, data)
		},
	}

	cmdGenerate.Flags().StringP(parameterSchema, "s", "", `Schema file`)
	cmdGenerate.Flags().StringSliceP(parameterData, "d", nil, `Data file`)
	cmdGenerate.Flags().StringP(parameterBase, "b", "", `Base path of files`)

	return cmdGenerate
}

func validateJsons(basepath, schemaFile string, dataFiles []string) error {

	if basepath != "" {
		err := os.Chdir(basepath)
		if err != nil {
			panic(err.Error())
		}
	}

	schema, err := loadJsonLoader(schemaFile)
	if err != nil {
		return err
	}

	documents, err := mergeAndLoadData(dataFiles)
	if err != nil {
		return err
	}

	validateJson(schema, documents)
	return nil
}

func readFile(filePath string) ([]byte, error) {

	return os.ReadFile(filePath)

}
func mergeAndLoadData(filePaths []string) (gojsonschema.JSONLoader, error) {

	if len(filePaths) == 1 {
		return loadJsonLoader(filePaths[0])
	}

	base := map[string]interface{}{}

	for _, filePath := range filePaths {
		currentMap := map[string]interface{}{}

		currentMap, err := loadFileAsMap(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s", filePath)
		}

		// Merge with the previous map
		base = mergeMaps(base, currentMap)
	}

	return gojsonschema.NewGoLoader(base), nil

}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func loadFileAsMap(path string) (map[string]interface{}, error) {

	var res map[string]interface{}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	dat, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	if strings.EqualFold(ext, ".yaml") || strings.EqualFold(ext, ".yml") {
		err = yaml.Unmarshal(dat, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	err = json.Unmarshal(dat, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func loadJsonLoader(path string) (gojsonschema.JSONLoader, error) {

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	if strings.EqualFold(ext, ".yaml") || strings.EqualFold(ext, ".yml") {
		dat, err := os.ReadFile(absPath)
		jsonData, err := yaml.YAMLToJSON(dat)
		if err != nil {
			return nil, err
		}
		return gojsonschema.NewBytesLoader(jsonData), nil
	}

	dataSource := fmt.Sprintf("file://%s", absPath)
	return gojsonschema.NewReferenceLoader(dataSource), nil
}

func validateJson(schemaLoader gojsonschema.JSONLoader, documentLoader gojsonschema.JSONLoader) {

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		fmt.Printf("The document is valid\n")
	} else {
		fmt.Printf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
	}
}
