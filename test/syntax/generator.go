package syntax

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/lucasjones/reggen"
	"github.com/synadia-io/connect/model"
)

const DefaultProduceSubject = "default.produce.subject"
const DefaultConsumeSubject = "default.consume.subject.>"

func generateDefaultInlet(component model.Component) (*model.Steps, error) {
	result := model.Steps{
		Source: &model.SourceStep{},
		Producer: &model.ProducerStep{
			Core: &model.ProducerStepCore{
				Subject: DefaultProduceSubject,
			},
		},
	}

	// Apply component defaults
	fields, err := getFields(component.Fields)
	if err != nil {
		return nil, err
	}

	result.Source.Type = component.Name
	result.Source.Config = fields

	return &result, nil
}

func generateDefaultOutlet(component model.Component) (*model.Steps, error) {
	result := model.Steps{
		Sink: &model.SinkStep{},

		Consumer: &model.ConsumerStep{
			Core: &model.ConsumerStepCore{
				Subject: DefaultConsumeSubject,
			},
		},
	}

	// Apply component defaults
	fields, err := getFields(component.Fields)
	if err != nil {
		return nil, err
	}

	result.Sink.Type = component.Name
	result.Sink.Config = fields

	return &result, nil
}

func getFields(fields []model.ComponentField) (map[string]any, error) {
	result := make(map[string]any)

	for _, f := range fields {
		v, set, err := getFieldValue(f)
		if err != nil {
			return nil, fmt.Errorf("failed to generate default for field %s: %w", f.Name, err)
		}
		if !set {
			continue
		}

		result[f.Name] = v
	}

	return result, nil
}

func getFieldValue(fld model.ComponentField) (any, bool, error) {
	// -- if the field is not required, skip it
	if !isRequired(fld) {
		return nil, false, nil
	}

	// -- if the field has a default value, use it
	if fld.Default != nil {
		return fld.Default, true, nil
	}

	// -- in all other cases, we need to generate a sensible value
	val, err := generateFieldValue(fld)
	if err != nil {
		return nil, false, err
	}

	return val, true, nil
}

func isRequired(fld model.ComponentField) bool {
	if fld.Optional == nil {
		return true
	}

	return !*fld.Optional
}

func generateFieldValue(fld model.ComponentField) (any, error) {
	switch fld.Kind {
	case model.ComponentFieldKindList:
		length := rand.IntN(10)
		result := make([]any, length)
		for i := 0; i < length; i++ {
			v, err := generateFieldScalarValue(fld)
			if err != nil {
				return nil, err
			}
			result[i] = v
		}

		return result, nil
	case model.ComponentFieldKindMap:
		length := rand.IntN(10)
		result := make(map[string]any, length)
		for i := 0; i < length; i++ {
			key := "key" + fmt.Sprint(i)
			v, err := generateFieldScalarValue(fld)
			if err != nil {
				return nil, err
			}
			result[key] = v
		}
		return result, nil

	case model.ComponentFieldKindScalar, "":
		return generateFieldScalarValue(fld)

	default:
		return nil, fmt.Errorf("unknown field kind %s", fld.Kind)
	}
}

func generateFieldScalarValue(fld model.ComponentField) (any, error) {
	switch fld.Type {
	case model.ComponentFieldTypeInt:
		return generateIntValue(fld)
	case model.ComponentFieldTypeBool:
		return rand.IntN(2)%2 == 0, nil
	case model.ComponentFieldTypeCondition:
		return "content().length() > 0", nil
	case model.ComponentFieldTypeExpression:
		return "root = {\"key\": \"value\"}", nil
	case model.ComponentFieldTypeString:
		return generateStringValue(fld)
	case model.ComponentFieldTypeObject:
		var flds []model.ComponentField
		for _, f := range fld.Fields {
			if f == nil {
				continue
			}

			flds = append(flds, *f)
		}

		return getFields(flds)
	case model.ComponentFieldTypeScanner:
		return `{"to_the_end": {}}`, nil
	default:
		return nil, fmt.Errorf("unknown field type %s", fld.Type)
	}
}

func generateIntValue(fld model.ComponentField) (any, error) {
	var constRange model.ComponentFieldConstraintsElemRange
	for _, c := range fld.Constraints {
		if c.Range != nil {
			constRange = *c.Range
			break
		}
	}

	minVal := 0
	maxVal := 100
	if constRange.Gt != nil {
		minVal = int(math.Ceil(*constRange.Gt)) + 1
	}
	if constRange.Gte != nil {
		minVal = int(math.Ceil(*constRange.Gte))
	}

	if constRange.Lt != nil {
		maxVal = int(math.Floor(*constRange.Lt)) - 1
	}
	if constRange.Lte != nil {
		maxVal = int(math.Floor(*constRange.Lte))
	}

	if minVal > maxVal {
		return 0, fmt.Errorf("invalid range: min %d > max %d", minVal, maxVal)
	}

	return minVal + rand.IntN(maxVal-minVal), nil
}

func generateStringValue(fld model.ComponentField) (any, error) {
	var constEnum []string
	var constPattern string

	// -- check for constraints
	for _, c := range fld.Constraints {
		if c.Enum != nil {
			constEnum = c.Enum
		}
		if c.Regex != nil {
			constPattern = *c.Regex
		}
	}

	// -- if enum is defined, pick a random value from it
	if len(constEnum) > 0 {
		return constEnum[rand.IntN(len(constEnum))], nil
	}

	// -- if pattern is defined, generate a value matching the pattern (not implemented)
	if constPattern != "" {
		return reggen.Generate(constPattern, 10)
	}

	// -- otherwise return a random string
	return "random-string" + fmt.Sprint(rand.IntN(10000)), nil
}
