package generators

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/juanvillacortac/rosetta/pkg/ast"
	"github.com/juanvillacortac/rosetta/pkg/types"
	"github.com/juanvillacortac/rosetta/pkg/utils"
)

type Generator func(root *ast.RootNode) ([]OutputFile, error)

type TypesMap map[types.IntrinsicType]string

type GenerateConfig struct {
	Name     string            `json:"name"`
	Template string            `json:"template"`
	Output   string            `json:"output"`
	Types    TypesMap          `json:"types"`
	Helpers  map[string]string `json:"helpers"`
}

func AdaptModel(models ast.ModelMap, typesMap TypesMap) ast.ModelMap {
	clone := make(ast.ModelMap)
	for _, m := range models {
		p := new(ast.Model)
		*p = *m
		clone[m.ModelName] = p
	}
	for k, m := range clone {
		for i, p := range m.Props {
			if t, ok := typesMap[types.IntrinsicType(p.Type)]; ok {
				clone[k].Props[i].Type = t
			}
		}
	}
	return clone
}

func Generate(root *ast.RootNode, options GenerateConfig) ([]OutputFile, error) {
	models := AdaptModel(root.Models, options.Types)
	reader, err := os.Open(options.Template)
	if err != nil {
		err = fmt.Errorf("failed to open %s, err %v", options.Template, err)
		return nil, err
	}
	defer reader.Close()

	buffer := bytes.Buffer{}
	buffer.ReadFrom(reader)
	if _, err := buffer.ReadFrom(reader); err != nil {
		return nil, err
	}
	t, err := template.New(options.Name).Funcs(templateHelpers(models, options)).Parse(buffer.String())
	if err != nil {
		panic(err)
	}
	files := make([]OutputFile, 0)
	for _, m := range models {
		deps := make([]string, 0)

		deps = models.GetModelDeps(m.ModelName, deps)

		writer := &strings.Builder{}
		err = t.Execute(writer, &struct {
			Deps  []string
			Root  *ast.RootNode
			Model *ast.Model
		}{
			Deps:  deps,
			Root:  root,
			Model: m,
		})
		if err != nil {
			return nil, err
		}
		filename := strings.ReplaceAll(options.Output, "[model]", m.ModelName)
		filename = strings.ReplaceAll(filename, "[Model]", strings.ToUpper(m.ModelName))
		filename = strings.ReplaceAll(filename, "[model_]", utils.ToSnakeCase(m.ModelName))
		filename = strings.ReplaceAll(filename, "[model-]", utils.ToKebabCase(m.ModelName))
		files = append(files, OutputFile{
			Filename: filename,
			Body:     writer.String(),
		})
	}
	return files, nil
}
