package gltf2

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/pkg/errors"
)

// if define Matrix, can't have TRS
type Node struct {
	Camera      *Camera
	Parent      *Node
	Children    []*Node
	Skin        *Skin
	Matrix      mgl32.Mat4
	Rotation    mgl32.Quat
	Scale       mgl32.Vec3
	Translation mgl32.Vec3
	Weights     []float32
	Mesh        *Mesh
	Name        string
	Extensions  *Extensions `json:"extensions,omitempty"`
	Extras      *Extras     `json:"extras,omitempty"`

	// None spec
	UserData interface{}
}

func (s *Node) GetExtension() *Extensions {
	return s.Extensions
}
func (s *Node) SetExtension(extensions *Extensions) {
	s.Extensions = extensions
}
func (s *Node) Transform() mgl32.Mat4 {
	if s.Matrix == mgl32.Ident4() {
		return mgl32.Translate3D(s.Translation[0], s.Translation[1], s.Translation[2]).
			Mul4(s.Rotation.Mat4()).
			Mul4(mgl32.Scale3D(s.Scale[0], s.Scale[1], s.Scale[2]))
	} else {
		return s.Matrix
	}
}

type SpecNode struct {
	Camera      *SpecGLTFID     `json:"camera"`
	Children    []SpecGLTFID    `json:"children"`    // unique, minItem(1)
	Skin        *SpecGLTFID     `json:"skin"`        // dependancy(Mesh)
	Matrix      *mgl32.Mat4     `json:"matrix"`      // default(mgl32.Ident4()), exclusive(Translation, Rotation, Scale)
	Rotation    *mgl32.Vec4     `json:"rotation"`    // default(mgl32.Vec4{0,0,0,1})
	Scale       *mgl32.Vec3     `json:"scale"`       // default(mgl32.Vec{1,1,1})
	Translation *mgl32.Vec3     `json:"translation"` // default(mgl32.Vec{0,0,0})
	Weights     []float32       `json:"weights"`     // minItem(1), dependancy(Mesh)
	Mesh        *SpecGLTFID     `json:"mesh"`        //
	Name        *string         `json:"name,omitempty"`
	Extensions  *SpecExtensions `json:"extensions,omitempty"`
	Extras      *Extras         `json:"extras,omitempty"`
}

func (s *SpecNode) SpecExtension() *SpecExtensions {
	return s.Extensions
}

func (s *SpecNode) Scheme() string {
	return SCHEME_NODE
}
func (s *SpecNode) Syntax(strictness Strictness, root Specifier, parent Specifier) error {
	switch strictness {
	case LEVEL3:
		fallthrough
	case LEVEL2:
		if s.Matrix != nil && (s.Translation != nil || s.Rotation != nil || s.Scale != nil) {
			return errors.Errorf("Node.Matrix exclusive(Translation, Rotation, Scale)")
		}
		fallthrough
	case LEVEL1:
		if is, i := isUniqueGLTFID(s.Children...); !is {
			return errors.Errorf("Node.Children not unique '%d'", i)
		}
		if s.Skin != nil && s.Mesh == nil {
			return errors.WithMessage(ErrorGLTFSpec, "Node.Skin dependancy(Mesh)")
		}
		if s.Weights != nil && s.Mesh == nil {
			return errors.WithMessage(ErrorGLTFSpec, "Node.Weights dependancy(Mesh)")
		}
	}
	return nil
}
func (s *SpecNode) To(ctx *parserContext) interface{} {
	res := new(Node)
	if s.Name != nil {
		res.Name = *s.Name
	}
	if s.Matrix == nil {
		res.Matrix = mgl32.Ident4()
	} else {
		res.Matrix = *s.Matrix
	}

	if s.Translation == nil {
		res.Translation = mgl32.Vec3{0, 0, 0}
	} else {
		res.Translation = *s.Translation
	}

	if s.Rotation == nil {
		res.Rotation = mgl32.QuatIdent()
	} else {
		res.Rotation = mgl32.QuatRotate(s.Rotation[3], s.Rotation.Vec3())
	}

	if s.Scale == nil {
		res.Scale = mgl32.Vec3{1, 1, 1}
	} else {
		res.Scale = *s.Scale
	}

	if s.Weights != nil {
		res.Weights = s.Weights
	}

	res.Extras = s.Extras
	return res
}
func (s *SpecNode) Link(Root *GLTF, parent interface{}, dst interface{}) error {
	if s.Camera != nil {
		if !inRange(*s.Camera, len(Root.Cameras)) {
			return errors.Errorf("Node.CameraSetting linking fail")
		}
		dst.(*Node).Camera = Root.Cameras[*s.Camera]
	}
	dst.(*Node).Children = make([]*Node, len(s.Children))
	for i, v := range s.Children {
		if !inRange(v, len(Root.Nodes)) {
			return errors.Errorf("Node.Children[%d] linking fail", i)
		}
		if findRecursiveLink(Root.Nodes[v], dst.(*Node)) {
			return errors.Errorf("Node.Children[%d] recursive link", i)
		}
		Root.Nodes[v].Parent = dst.(*Node)
		dst.(*Node).Children[i] = Root.Nodes[v]
	}
	if s.Mesh != nil {
		if !inRange(*s.Mesh, len(Root.Meshes)) {
			return errors.Errorf("Node.Mesh linking fail")
		}
		dst.(*Node).Mesh = Root.Meshes[*s.Mesh]
	}
	if s.Skin != nil {
		if !inRange(*s.Skin, len(Root.Skins)) {
			return errors.Errorf("Node.Skin linking fail")
		}
		dst.(*Node).Skin = Root.Skins[*s.Skin]
	}
	return nil
}

func findRecursiveLink(child, parent *Node) bool {
	if parent == nil {
		return false
	}
	if parent == child {
		return true
	}
	return findRecursiveLink(child, parent.Parent)
}
