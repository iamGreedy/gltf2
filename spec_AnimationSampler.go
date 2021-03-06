package gltf2

import (
	"github.com/pkg/errors"
)

type AnimationSampler struct {
	Input         *Accessor
	Interpolation Interpolation
	Output        *Accessor
	Extensions    *Extensions
	Extras        *Extras
	// None spec
	UserData interface{}
}

func (s *AnimationSampler) GetExtension() *Extensions {
	return s.Extensions
}

func (s *AnimationSampler) SetExtension(extensions *Extensions) {
	s.Extensions = extensions
}

type SpecAnimationSampler struct {
	Input         *SpecGLTFID     `json:"input"`         // required
	Interpolation *Interpolation  `json:"interpolation"` // default(LINEAR)
	Output        *SpecGLTFID     `json:"output"`        // required, AnimationSampler.Output -> Accessor.ComponentType must(FLOAT or normalized integer)
	Extensions    *SpecExtensions `json:"extensions,omitempty"`
	Extras        *Extras         `json:"extras,omitempty"`
}

func (s *SpecAnimationSampler) SpecExtension() *SpecExtensions {
	return s.Extensions
}
func (s *SpecAnimationSampler) Scheme() string {
	return SCHEME_ANIMATION_SAMPLER
}
func (s *SpecAnimationSampler) Syntax(strictness Strictness, root Specifier, parent Specifier) error {
	switch strictness {
	case LEVEL3:
		fallthrough
	case LEVEL2:
		fallthrough
	case LEVEL1:
		if s.Input == nil {
			return errors.Errorf("AnimationSampler.Input required")
		}
		if s.Output == nil {
			return errors.Errorf("AnimationSampler.Output required")
		}
		if s.Output != nil && inRange(*s.Output, len(root.(*SpecGLTF).Accessors)) {

			if acc := root.(*SpecGLTF).Accessors[*s.Output]; !((acc.ComponentType != nil && *acc.ComponentType == FLOAT) ||
				(acc.Normalized != nil && *acc.Normalized)) {
				return errors.Errorf("AnimationSampler.Output -> Accessor.ComponentType must(FLOAT or normalized integer)")
			}
		}
	}
	return nil
}
func (s *SpecAnimationSampler) To(ctx *parserContext) interface{} {
	res := new(AnimationSampler)
	if s.Interpolation == nil {
		res.Interpolation = LINEAR
	} else {
		res.Interpolation = *s.Interpolation
	}
	res.Extras = s.Extras
	return res
}
func (s *SpecAnimationSampler) Link(Root *GLTF, parent interface{}, dst interface{}) error {
	if !inRange(*s.Input, len(Root.Accessors)) {
		return errors.Errorf("AnimationSampler.Input linking fail")
	}
	dst.(*AnimationSampler).Input = Root.Accessors[*s.Input]
	if !inRange(*s.Output, len(Root.Accessors)) {
		return errors.Errorf("AnimationSampler.Output linking fail")
	}
	dst.(*AnimationSampler).Output = Root.Accessors[*s.Output]
	return nil
}
