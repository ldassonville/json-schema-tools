package registry

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func NewRelocator(location string, execDir string, targetDir string) *relocator {

	return &relocator{
		catalog:     make(map[string]string),
		execDir:     execDir,
		destDir:     targetDir,
		newLocation: location,
	}
}

type relocator struct {
	execDir     string
	destDir     string
	newLocation string
	catalog     map[string]string
}

func (r *relocator) Relocate(file string) {

	err := os.Chdir(r.execDir)
	if err != nil {
		panic(err.Error())
	}

	r.relocateJsonFile(file)
}

func (r *relocator) relocateJsonFile(file string) {

	jsonFile, err := os.Open(file)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	vc := &VisitorContext{
		File: file,
	}
	data, err := r.Load(byteValue, vc)

	if data != nil {
		_, _, targetFile, err := r.resolveLocation(file, vc)

		parent := filepath.Dir(targetFile)

		if strings.HasPrefix(parent, r.destDir) {

			os.MkdirAll(parent, 0755)

			log.Info().Msgf("Writing file : %s", file)
			err = os.WriteFile(targetFile, data, 0777)
			// handle this error
			if err != nil {

				log.Err(err).Msgf("Fail to write file")
			}
		} else {
			log.Panic().Msgf("trying to write out the target")
		}
	}
}

func (r *relocator) Load(byteValue []byte, vc *VisitorContext) ([]byte, error) {

	var err error
	var schema = map[string]interface{}{}

	err = json.Unmarshal(byteValue, &schema)
	if err != nil {
		return nil, err
	}

	return r.visit(schema, vc)

}

func (r *relocator) resolveLocation(location string, vc *VisitorContext) (absPath string, newRef string, newPath string, err error) {

	location = strings.TrimPrefix(location, "file://")

	// Resolve the absolute path of the file
	// relatively to the given base exec directory
	absPath, err = filepath.Abs(location)
	if err != nil {
		log.Err(err).Msgf("fail to convert abs path")
		return "", "", "", err
	}

	// Resolve the relative position of the file
	// in the exec directory
	execRelPath, err := filepath.Rel(r.execDir, absPath)
	if err != nil {
		log.Err(err).Msgf("fail to convert to relative path")
		return "", "", "", err
	}

	// Get the absolute path of the given file in the
	// destination directory
	newPath = filepath.Join(r.destDir, execRelPath)
	newPath, err = filepath.Abs(newPath)
	if err != nil {
		log.Err(err).Msgf("fail to convert newPath to abs path")
		return "", "", "", err
	}

	refRelPath, err := filepath.Rel(filepath.Dir(vc.File), absPath)
	if err != nil {
		log.Err(err).Msgf("fail to determine relative path from the current file")
		return "", "", "", err
	}

	newRef = filepath.Join(r.newLocation, refRelPath)

	return
}

func (r *relocator) resolveRef(ref string, vc *VisitorContext) (string, error) {

	if strings.HasPrefix(ref, "file://") {

		refParts := strings.Split(ref, "#")
		location := refParts[0]
		resource := refParts[1]

		absPath, newRef, _, err := r.resolveLocation(location, vc)
		if err != nil {
			log.Err(err).Msgf("fail to convert abs path")
			return "", err
		}

		if _, found := r.catalog[absPath]; !found {
			r.catalog[absPath] = "ok"
			r.relocateJsonFile(absPath)
		}

		return fmt.Sprintf("%s#%s", newRef, resource), nil
	}

	return ref, nil
}

type VisitorContext struct {
	File string
}

func (r *relocator) visitAllOf(definition map[string]interface{}, vc *VisitorContext) {
	if allOf, haveAllOf := definition["allOf"]; haveAllOf {

		if allOfSlice, ok := allOf.([]interface{}); ok {
			for _, allOfItem := range allOfSlice {

				if allOfItemMap, ok := allOfItem.(map[string]interface{}); ok {
					if ref, haveRef := allOfItemMap["$ref"]; haveRef {

						resolvedRef, err := r.resolveRef(ref.(string), vc)
						if err != nil {
							log.Err(err).Msgf("Fail to resolve ref %s", resolvedRef)
							continue
						}
						allOfItemMap["$ref"] = resolvedRef
					}
				}
			}
		}
	}
}
func (r *relocator) visitProperties(definition map[string]interface{}, vc *VisitorContext) {

	if properties, haveProperties := definition["properties"]; haveProperties {

		if propertiesMap, ok := properties.(map[string]interface{}); ok {

			for propertyKey, propertyConfig := range propertiesMap {

				log.Debug().Msgf("processing key %s", propertyKey)

				if propertyConfigMap, ok := propertyConfig.(map[string]interface{}); ok {

					if ref, haveRef := propertyConfigMap["$ref"]; haveRef {

						resolvedRef, err := r.resolveRef(ref.(string), vc)
						if err != nil {
							log.Err(err).Msgf("Fail to resolve ref %s", resolvedRef)
							continue
						}
						propertyConfigMap["$ref"] = resolvedRef
					}
				}
			}
		}
	}
}

func (r *relocator) visit(schema map[string]interface{}, vc *VisitorContext) ([]byte, error) {

	// Resolve properties
	r.visitProperties(schema, vc)
	r.visitAllOf(schema, vc)

	if definitions, haveDefinition := schema["$def"]; haveDefinition {
		if definitionMap, ok := definitions.(map[string]interface{}); ok {

			for definitionKey, definition := range definitionMap {

				log.Debug().Msgf("processing definition %s", definitionKey)

				defMap, _ := definition.(map[string]interface{})

				r.visitProperties(defMap, vc)
				r.visitAllOf(defMap, vc)
			}
		}
	}

	data, err := json.MarshalIndent(schema, "", "   ")

	if err != nil {
		log.Err(err).Msgf("Fail to generate json")
		return nil, err
	}

	return data, nil
}
