package openapi3

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

type SecuritySchemes map[string]*SecuritySchemeRef

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (s SecuritySchemes) JSONLookup(token string) (interface{}, error) {
	ref, ok := s[token]
	if ref == nil || ok == false {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

var _ jsonpointer.JSONPointable = (*SecuritySchemes)(nil)

// SecurityScheme is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#securitySchemeObject
type SecurityScheme struct {
	ExtensionProps `json:"-" yaml:"-"`

	Type             string      `json:"type,omitempty" yaml:"type,omitempty"`
	Description      string      `json:"description,omitempty" yaml:"description,omitempty"`
	Name             string      `json:"name,omitempty" yaml:"name,omitempty"`
	In               string      `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`
	OpenIdConnectUrl string      `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

func NewSecurityScheme() *SecurityScheme {
	return &SecurityScheme{}
}

func NewCSRFSecurityScheme() *SecurityScheme {
	return &SecurityScheme{
		Type: "apiKey",
		In:   "header",
		Name: "X-XSRF-TOKEN",
	}
}

func NewOIDCSecurityScheme(oidcUrl string) *SecurityScheme {
	return &SecurityScheme{
		Type:             "openIdConnect",
		OpenIdConnectUrl: oidcUrl,
	}
}

func NewJWTSecurityScheme() *SecurityScheme {
	return &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
}

// MarshalJSON returns the JSON encoding of SecurityScheme.
func (ss *SecurityScheme) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(ss)
}

// UnmarshalJSON sets SecurityScheme to a copy of data.
func (ss *SecurityScheme) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, ss)
}

func (ss *SecurityScheme) WithType(value string) *SecurityScheme {
	ss.Type = value
	return ss
}

func (ss *SecurityScheme) WithDescription(value string) *SecurityScheme {
	ss.Description = value
	return ss
}

func (ss *SecurityScheme) WithName(value string) *SecurityScheme {
	ss.Name = value
	return ss
}

func (ss *SecurityScheme) WithIn(value string) *SecurityScheme {
	ss.In = value
	return ss
}

func (ss *SecurityScheme) WithScheme(value string) *SecurityScheme {
	ss.Scheme = value
	return ss
}

func (ss *SecurityScheme) WithBearerFormat(value string) *SecurityScheme {
	ss.BearerFormat = value
	return ss
}

// Validate returns an error if SecurityScheme does not comply with the OpenAPI spec.
func (ss *SecurityScheme) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	hasIn := false
	hasBearerFormat := false
	hasFlow := false
	switch ss.Type {
	case "apiKey":
		hasIn = true
	case "http":
		scheme := ss.Scheme
		switch scheme {
		case "bearer":
			hasBearerFormat = true
		case "basic", "negotiate", "digest":
		default:
			return fmt.Errorf("security scheme of type 'http' has invalid 'scheme' value %q", scheme)
		}
	case "oauth2":
		hasFlow = true
	case "openIdConnect":
		if ss.OpenIdConnectUrl == "" {
			return fmt.Errorf("no OIDC URL found for openIdConnect security scheme %q", ss.Name)
		}
	default:
		return fmt.Errorf("security scheme 'type' can't be %q", ss.Type)
	}

	// Validate "in" and "name"
	if hasIn {
		switch ss.In {
		case "query", "header", "cookie":
		default:
			return fmt.Errorf("security scheme of type 'apiKey' should have 'in'. It can be 'query', 'header' or 'cookie', not %q", ss.In)
		}
		if ss.Name == "" {
			return errors.New("security scheme of type 'apiKey' should have 'name'")
		}
	} else if len(ss.In) > 0 {
		return fmt.Errorf("security scheme of type %q can't have 'in'", ss.Type)
	} else if len(ss.Name) > 0 {
		return errors.New("security scheme of type 'apiKey' can't have 'name'")
	}

	// Validate "format"
	// "bearerFormat" is an arbitrary string so we only check if the scheme supports it
	if !hasBearerFormat && len(ss.BearerFormat) > 0 {
		return fmt.Errorf("security scheme of type %q can't have 'bearerFormat'", ss.Type)
	}

	// Validate "flow"
	if hasFlow {
		flow := ss.Flows
		if flow == nil {
			return fmt.Errorf("security scheme of type %q should have 'flows'", ss.Type)
		}
		if err := flow.Validate(ctx); err != nil {
			return fmt.Errorf("security scheme 'flow' is invalid: %w", err)
		}
	} else if ss.Flows != nil {
		return fmt.Errorf("security scheme of type %q can't have 'flows'", ss.Type)
	}
	return nil
}

// OAuthFlows is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#oauthFlowsObject
type OAuthFlows struct {
	ExtensionProps `json:"-" yaml:"-"`

	Implicit          *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

type oAuthFlowType int

const (
	oAuthFlowTypeImplicit oAuthFlowType = iota
	oAuthFlowTypePassword
	oAuthFlowTypeClientCredentials
	oAuthFlowAuthorizationCode
)

// MarshalJSON returns the JSON encoding of OAuthFlows.
func (flows *OAuthFlows) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(flows)
}

// UnmarshalJSON sets OAuthFlows to a copy of data.
func (flows *OAuthFlows) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, flows)
}

// Validate returns an error if OAuthFlows does not comply with the OpenAPI spec.
func (flows *OAuthFlows) Validate(ctx context.Context, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	if v := flows.Implicit; v != nil {
		if err := v.validate(ctx, oAuthFlowTypeImplicit, opts...); err != nil {
			return fmt.Errorf("the OAuth flow 'implicit' is invalid: %w", err)
		}
	}
	if v := flows.Password; v != nil {
		if err := v.validate(ctx, oAuthFlowTypePassword, opts...); err != nil {
			return fmt.Errorf("the OAuth flow 'password' is invalid: %w", err)
		}
	}
	if v := flows.ClientCredentials; v != nil {
		if err := v.validate(ctx, oAuthFlowTypeClientCredentials, opts...); err != nil {
			return fmt.Errorf("the OAuth flow 'clientCredentials' is invalid: %w", err)
		}
	}
	if v := flows.AuthorizationCode; v != nil {
		if err := v.validate(ctx, oAuthFlowAuthorizationCode, opts...); err != nil {
			return fmt.Errorf("the OAuth flow 'authorizationCode' is invalid: %w", err)
		}
	}
	return nil
}

// OAuthFlow is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#oauthFlowObject
type OAuthFlow struct {
	ExtensionProps `json:"-" yaml:"-"`

	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes" yaml:"scopes"`
}

// MarshalJSON returns the JSON encoding of OAuthFlow.
func (flow *OAuthFlow) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(flow)
}

// UnmarshalJSON sets OAuthFlow to a copy of data.
func (flow *OAuthFlow) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, flow)
}

// Validate returns an error if OAuthFlows does not comply with the OpenAPI spec.
func (flow *OAuthFlow) Validate(ctx context.Context, opts ...ValidationOption) error {
	// ctx = WithValidationOptions(ctx, opts...)

	if v := flow.RefreshURL; v != "" {
		if _, err := url.Parse(v); err != nil {
			return fmt.Errorf("field 'refreshUrl' is invalid: %w", err)
		}
	}

	if v := flow.Scopes; len(v) == 0 {
		return errors.New("field 'scopes' is empty or missing")
	}

	return nil
}

func (flow *OAuthFlow) validate(ctx context.Context, typ oAuthFlowType, opts ...ValidationOption) error {
	ctx = WithValidationOptions(ctx, opts...)

	typeIn := func(types ...oAuthFlowType) bool {
		for _, ty := range types {
			if ty == typ {
				return true
			}
		}
		return false
	}

	if in := typeIn(oAuthFlowTypeImplicit, oAuthFlowAuthorizationCode); true {
		switch {
		case flow.AuthorizationURL == "" && in:
			return errors.New("field 'authorizationUrl' is empty or missing")
		case flow.AuthorizationURL != "" && !in:
			return errors.New("field 'authorizationUrl' should not be set")
		case flow.AuthorizationURL != "":
			if _, err := url.Parse(flow.AuthorizationURL); err != nil {
				return fmt.Errorf("field 'authorizationUrl' is invalid: %w", err)
			}
		}
	}

	if in := typeIn(oAuthFlowTypePassword, oAuthFlowTypeClientCredentials, oAuthFlowAuthorizationCode); true {
		switch {
		case flow.TokenURL == "" && in:
			return errors.New("field 'tokenUrl' is empty or missing")
		case flow.TokenURL != "" && !in:
			return errors.New("field 'tokenUrl' should not be set")
		case flow.TokenURL != "":
			if _, err := url.Parse(flow.TokenURL); err != nil {
				return fmt.Errorf("field 'tokenUrl' is invalid: %w", err)
			}
		}
	}

	return flow.Validate(ctx, opts...)
}
