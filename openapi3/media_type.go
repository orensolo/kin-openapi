package openapi3

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

// MediaType is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#mediaTypeObject
type MediaType struct {
	ExtensionProps `json:"-" yaml:"-"`

	Schema   *SchemaRef           `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}          `json:"example,omitempty" yaml:"example,omitempty"`
	Examples Examples             `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]*Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

var _ jsonpointer.JSONPointable = (*MediaType)(nil)

func NewMediaType() *MediaType {
	return &MediaType{}
}

func (mediaType *MediaType) WithSchema(schema *Schema) *MediaType {
	if schema == nil {
		mediaType.Schema = nil
	} else {
		mediaType.Schema = &SchemaRef{Value: schema}
	}
	return mediaType
}

func (mediaType *MediaType) WithSchemaRef(schema *SchemaRef) *MediaType {
	mediaType.Schema = schema
	return mediaType
}

func (mediaType *MediaType) WithExample(name string, value interface{}) *MediaType {
	example := mediaType.Examples
	if example == nil {
		example = make(map[string]*ExampleRef)
		mediaType.Examples = example
	}
	example[name] = &ExampleRef{
		Value: NewExample(value),
	}
	return mediaType
}

func (mediaType *MediaType) WithEncoding(name string, enc *Encoding) *MediaType {
	encoding := mediaType.Encoding
	if encoding == nil {
		encoding = make(map[string]*Encoding)
		mediaType.Encoding = encoding
	}
	encoding[name] = enc
	return mediaType
}

// MarshalJSON returns the JSON encoding of MediaType.
func (mediaType *MediaType) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(mediaType)
}

// UnmarshalJSON sets MediaType to a copy of data.
func (mediaType *MediaType) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, mediaType)
}

// Validate returns an error if MediaType does not comply with the OpenAPI spec.
func (mediaType *MediaType) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	if mediaType == nil {
		return nil
	}
	if schema := mediaType.Schema; schema != nil {
		if err := schema.Validate(ctx); err != nil {
			return err
		}

		if mediaType.Example != nil && mediaType.Examples != nil {
			return errors.New("example and examples are mutually exclusive")
		}

		if vo := getValidationOptions(ctx); vo.ExamplesValidationDisabled {
			return nil
		}

		if example := mediaType.Example; example != nil {
			if err := validateExampleValue(ctx, example, schema.Value); err != nil {
				return fmt.Errorf("invalid example: %w", err)
			}
		}

		if examples := mediaType.Examples; examples != nil {
			names := make([]string, 0, len(examples))
			for name := range examples {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, k := range names {
				v := examples[k]
				if err := v.Validate(ctx); err != nil {
					return fmt.Errorf("example %s: %w", k, err)
				}
				if err := validateExampleValue(ctx, v.Value.Value, schema.Value); err != nil {
					return fmt.Errorf("example %s: %w", k, err)
				}
			}
		}
	}

	return nil
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (mediaType MediaType) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "schema":
		if mediaType.Schema != nil {
			if mediaType.Schema.Ref != "" {
				return &Ref{Ref: mediaType.Schema.Ref}, nil
			}
			return mediaType.Schema.Value, nil
		}
	case "example":
		return mediaType.Example, nil
	case "examples":
		return mediaType.Examples, nil
	case "encoding":
		return mediaType.Encoding, nil
	}
	v, _, err := jsonpointer.GetForToken(mediaType.ExtensionProps, token)
	return v, err
}
