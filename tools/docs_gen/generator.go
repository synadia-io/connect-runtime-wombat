package main

import (
	"fmt"
	"strings"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/tools/docs_gen/model"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"gopkg.in/yaml.v3"
)

var StatusMapping = map[string]model.ComponentStatusSpec{
	"beta":         model.ComponentStatusSpecPreview,
	"stable":       model.ComponentStatusSpecStable,
	"experimental": model.ComponentStatusSpecExperimental,
	"deprecated":   model.ComponentStatusSpecDeprecated,
}

var FieldTypes = map[string]model.FieldSpecType{
	"string":     model.FieldSpecTypeString,
	"object":     model.FieldSpecTypeObject,
	"array":      model.FieldSpecTypeObject, // Arrays are represented as objects with kind:list
	"bool":       model.FieldSpecTypeBool,
	"int":        model.FieldSpecTypeInt,
	"scanner":    model.FieldSpecTypeScanner,
	"expression": model.FieldSpecTypeExpression,
	"condition":  model.FieldSpecTypeCondition,
	"input":      model.FieldSpecTypeObject, // Inputs are complex objects
}

func Generate(data service.TemplateDataPlugin, componentType string) (*model.ComponentSpec, error) {
	status, fnd := StatusMapping[data.Status]
	if !fnd {
		status = model.ComponentStatusSpecExperimental
	}

	// -- convert the flat tree structure into a tree
	ft := fieldTree(data.Fields)
	fields := generateFields(ft.children)
	npFields := make([]model.FieldSpec, 0, len(fields))
	for _, f := range fields {
		if f != nil {
			npFields = append(npFields, *f)
		}
	}

	description := data.Summary
	if data.Description != "" {
		description += "\n\n"
		description += data.Description
	}

	// Map component type to ComponentKindSpec
	var kind model.ComponentKindSpec
	switch componentType {
	case "input":
		kind = model.ComponentKindSpecSource
	case "output":
		kind = model.ComponentKindSpecSink
	case "scanner":
		kind = model.ComponentKindSpecScanner
	default:
		// If component type is not recognized, return an error
		return nil, fmt.Errorf("unknown component type: %s", componentType)
	}

	return &model.ComponentSpec{
		Description: description,
		Name:        data.Name,
		Label:       toHumanReadableLabel(data.Name),
		Status:      status,
		Kind:        kind,
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

		// Determine the correct kind based on type
		var kind model.FieldSpecKind
		switch field.Field.Type {
		case "array":
			kind = model.FieldSpecKindList
		case "object":
			// Objects with defined fields are scalar, generic objects are maps
			if len(field.children) > 0 {
				kind = model.FieldSpecKindScalar
			} else {
				kind = model.FieldSpecKindMap
			}
		default:
			kind = model.FieldSpecKindScalar
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
			Kind:        kind,
			Label:       toHumanReadableFieldLabel(field.Name),
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
