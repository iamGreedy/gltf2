package gltf2

import (
	"github.com/pkg/errors"
)

type TextureInfo struct {
	Index      *Texture
	TexCoord   IndexTexCoord
	Extensions *Extensions
	Extras     *Extras
}

type SpecTextureInfo struct {
	Index      *SpecGLTFID    `json:"index"`    // required
	TexCoord   *IndexTexCoord `json:"texCoord"` // default(0), minimum(0)
	Extensions *Extensions    `json:"extensions,omitempty"`
	Extras     *Extras        `json:"extras,omitempty"`
}

func (s *SpecTextureInfo) Scheme() string {
	return SCHEME_TEXTURE_INFO
}

func (s *SpecTextureInfo) Syntax(strictness Strictness, root interface{}) error {
	switch strictness {
	case LEVEL3:
		fallthrough
	case LEVEL2:
		fallthrough
	case LEVEL1:
		if s.Index == nil {
			return errors.WithMessage(ErrorGLTFSpec, "TextureInfo.Index required")
		}
	}
	return nil
}

func (s *SpecTextureInfo) To(ctx *parserContext) interface{} {
	res := new(TextureInfo)
	if s.TexCoord == nil {
		res.TexCoord = TexCoord0
	} else {
		res.TexCoord = *s.TexCoord
	}
	res.Extras = s.Extras
	res.Extensions = s.Extensions
	return res
}

func (s *SpecTextureInfo) Link(Root *GLTF, parent interface{}, dst interface{}) error {
	if !inRange(*s.Index, len(Root.Textures)) {
		return errors.Errorf("TextureInfo.Index linking fail")
	}
	dst.(*TextureInfo).Index = Root.Textures[*s.Index]
	return nil
}
