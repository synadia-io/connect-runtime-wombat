package main

import (
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/model"
	"gopkg.in/yaml.v3"
	"strings"
)

var StatusMapping = map[string]model.ComponentStatusSpec{
	"beta":         model.ComponentStatusSpecPreview,
	"stable":       model.ComponentStatusSpecStable,
	"experimental": model.ComponentStatusSpecExperimental,
	"deprecated":   model.ComponentStatusSpecDeprecated,
}

var FieldTypes = map[string]model.FieldSpecType{
	"string": model.FieldSpecTypeString,
	"object": model.FieldSpecTypeObject,
}

func Generate(data service.TemplateDataPlugin) (*model.ComponentSpec, error) {
	status, fnd := StatusMapping[data.Status]
	if !fnd {
		status = model.ComponentStatusSpecExperimental
	}

	// -- convert the flat tree structure into a tree
	ft := fieldTree(data.Fields)
	fields := generateFields(ft.children)
	npFields := make([]model.FieldSpec, 0, len(fields))
	for _, f := range fields {
		npFields = append(npFields, *f)
	}

	description := data.Summary
	if data.Description != "" {
		description += "\n\n"
		description += data.Description
	}

	return &model.ComponentSpec{
		Description: description,
		Name:        data.Name,
		Label:       data.Name,
		Status:      status,
		Fields:      npFields,
		Icon:        nil,
	}, nil
}

type container struct {
	children []*fieldContainer
}

func (c *container) MustChild(name string) *fieldContainer {
	for _, child := range c.children {
		if child.Name == name {
			return child
		}
	}

	result := &fieldContainer{
		Name: name,
	}

	c.children = append(c.children, result)

	return result
}

type fieldContainer struct {
	container
	Name  string
	Field service.TemplateDataPluginField
}

func (ftc *fieldContainer) Traverse(path []string) *fieldContainer {
	if len(path) == 0 {
		return ftc
	}

	name := strings.TrimSuffix(path[0], "[]")

	return ftc.MustChild(name).Traverse(path[1:])
}

func (ftc *fieldContainer) AsYaml() *yaml.Node {
	result := &yaml.Node{
		Kind: yaml.SequenceNode,
	}

	for _, child := range ftc.children {
		result.Content = append(result.Content, child.AsYaml())
	}

	return result
}

type componentConfig struct {
	container
}

func (cc componentConfig) Get(path []string) *fieldContainer {
	if len(path) == 0 {
		return nil
	}

	name := strings.TrimSuffix(path[0], "[]")
	return cc.MustChild(name).Traverse(path[1:])
}

func (cc *componentConfig) AsYaml() *yaml.Node {
	result := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	for _, child := range cc.children {
		result.Content = append(result.Content, child.AsYaml())
	}

	return result
}

func fieldTree(fields []service.TemplateDataPluginField) componentConfig {
	return componentConfig{
		container{
			children: fieldTreeElement("", fields),
		},
	}
}

func fieldTreeElement(prefix string, fields []service.TemplateDataPluginField) []*fieldContainer {
	var result []*fieldContainer

	var level int
	if prefix == "" {
		level = 1
	} else {
		level = len(strings.Split(prefix, ".")) + 1
	}

	for _, f := range fields {
		if f.FullName == prefix {
			continue
		}

		np := strings.Split(f.FullName, ".")

		// -- continue if we are not at the right level
		if len(np) != level {
			continue
		}

		// -- continue if the field is not a child of the prefix
		if !strings.HasPrefix(f.FullName, prefix) {
			continue
		}

		result = append(result, &fieldContainer{
			container: container{
				children: fieldTreeElement(f.FullName, fields),
			},
			Name:  strings.TrimSuffix(np[len(np)-1], "[]"),
			Field: f,
		})
	}

	return result
}

func generateFields(fields []*fieldContainer) []*model.FieldSpec {
	var results []*model.FieldSpec

	for _, field := range fields {
		ft, fnd := FieldTypes[field.Field.Type]
		if !fnd {
			ft = model.FieldSpecType(field.Field.Type)
		}

		rf := &model.FieldSpec{
			Name:        field.Name,
			Type:        ft,
			Path:        &field.Field.FullName,
			Description: &field.Field.Description,
			Constraints: nil,
			Default:     field.Field.DefaultMarshalled,
			Examples:    field.Field.Examples,
			Fields:      nil,
			Kind:        model.FieldSpecKindScalar,
			Label:       field.Name,
			Optional:    utils.Ptr(false),
			RenderHint:  nil,
		}

		if field.Field.IsSecret {
			rf.Secret = utils.Ptr(true)
		}

		if len(field.Field.Options) > 0 {
			rf.Constraints = append(rf.Constraints, model.ConstraintSpec{Enum: field.Field.Options})
		}

		if field.Field.DefaultMarshalled != "" {
			rf.Optional = utils.Ptr(true)
		}

		if len(field.children) > 0 {
			rf.Fields = generateFields(field.children)
		}

		results = append(results, rf)
	}

	return results
}
