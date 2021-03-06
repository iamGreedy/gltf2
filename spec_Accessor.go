package gltf2

import (
	"github.com/pkg/errors"
	"reflect"
	"unsafe"
)

type Accessor struct {
	BufferView    *BufferView
	ByteOffset    int
	Count         int
	Normalized    bool
	Type          AccessorType
	ComponentType ComponentType
	Max           []float32
	Min           []float32
	// TODO : Sparse        *AccessorSparse `json:"sparse,omitempty"`
	Name       string
	Extensions *Extensions
	Extras     *Extras
	// None spec
	UserData interface{}
}

func (s *Accessor) GetExtension() *Extensions {
	return s.Extensions
}

func (s *Accessor) SetExtension(extensions *Extensions) {
	s.Extensions = extensions
}

// [ Unsafe ] 		: careful to use
// [ Reflect ] 		: using reflect, it can be cause performance issue
//
// out_ptrslice 	: Pointer Slice Type for reading accessor
// typeSafety 		: typeSafety option, if enable, checking type safety by using reflect
//
// ex) data, err := <Accessor>.SliceMapping(new([][3]float), true)
//     slice := data.([][3]float)
func (s *Accessor) RawMap() ([]byte, error) {
	bts, err := s.BufferView.Load()
	if err != nil {
		return nil, err
	}
	bts = bts[s.ByteOffset:s.ByteOffset + s.Count * s.Type.Count() * s.ComponentType.Size()]
	return bts, nil
}
func (s *Accessor) SliceMapping(out_ptrslice interface{}, typeSafety, componentSafety bool) (interface{}, error) {
	if s.BufferView.ByteStride != 0 && s.BufferView.ByteStride != s.ComponentType.Size()*s.Type.Count() {
		return nil, errors.New("SliceMapping ByteStride support not available")
	}
	bts, err := s.RawMap()
	if err != nil {
		return nil, err
	}
	if len(bts) < 1 {
		return reflect.ValueOf(out_ptrslice).Elem().Interface(), nil
	}
	//
	tp := reflect.TypeOf(out_ptrslice)
	if tp.Kind() != reflect.Ptr && tp.Elem().Kind() == reflect.Slice {
		return nil, errors.New("out_ptrslice must be pointer slice")
	}
	// element type check
	var elemType = tp.Elem().Elem()
	var ctType = elemType
	// Check slice safety
	if typeSafety {
		switch s.Type {
		case SCALAR:
			if elemType.Kind() == reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must not be Array", elemType)
			}
			ctType = elemType
		case VEC2:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 2 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 2", elemType)
			}
			ctType = elemType.Elem()
		case VEC3:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 3 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 3", elemType)
			}
			ctType = elemType.Elem()
		case VEC4:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 4 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 4", elemType)
			}
			ctType = elemType.Elem()
		case MAT2:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 4 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 4", elemType)
			}
			ctType = elemType.Elem()
		case MAT3:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 9 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 9", elemType)
			}
			ctType = elemType.Elem()
		case MAT4:
			if elemType.Kind() != reflect.Array {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be Array", elemType)
			}
			if elemType.Len() != 16 {
				return nil, errors.Errorf("out_ptrslice elements type '%s' Array size must be 16", elemType)
			}
			ctType = elemType.Elem()
		}
	}
	if componentSafety {
		switch s.ComponentType {
		case BYTE:
			if !inKind(ctType.Kind(), reflect.Int8) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be int8", elemType)
			}
		case UNSIGNED_BYTE:
			if !inKind(ctType.Kind(), reflect.Uint8) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be uint8", elemType)
			}
		case SHORT:
			if !inKind(ctType.Kind(), reflect.Int16) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be int16", elemType)
			}
		case UNSIGNED_SHORT:
			if !inKind(ctType.Kind(), reflect.Uint16) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be uint16", elemType)
			}
		case UNSIGNED_INT:
			if !inKind(ctType.Kind(), reflect.Uint32) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be uint32", elemType)
			}
		case FLOAT:
			if !inKind(ctType.Kind(), reflect.Float32) {
				return nil, errors.Errorf("out_ptrslice elements type '%s' must be uint32", elemType)
			}
		}
	}
	//
	vl := reflect.ValueOf(out_ptrslice)
	header := (*reflect.SliceHeader)(unsafe.Pointer(vl.Pointer()))
	header.Data = uintptr(unsafe.Pointer(&bts[0]))
	header.Len = s.Count
	if !typeSafety {
		header.Len *= s.Type.Count()
	}
	header.Cap = header.Len
	return vl.Elem().Interface(), nil
}
func (s *Accessor) MustSliceMapping(out_ptrslice interface{}, typeSafety, componentSafety bool) interface{} {
	i, err := s.SliceMapping(out_ptrslice, typeSafety, componentSafety)
	if err != nil {
		panic(err)
	}
	return i
}

type SpecAccessor struct {
	BufferView    *SpecGLTFID     `json:"bufferView,omitempty"` //
	ByteOffset    *int            `json:"byteOffset,omitempty"` // default(0), minimum(0), dependency(bufferView)
	ComponentType *ComponentType  `json:"componentType"`        // required
	Normalized    *bool           `json:"normalized,omitempty"` // default(false)
	Count         *int            `json:"count"`                // required, minimum(1)
	Type          *AccessorType   `json:"type"`                 // required
	Max           []float32       `json:"max,omitempty"`        // rangeitem(1, 16)
	Min           []float32       `json:"min,omitempty"`        // rangeitem(1, 16)
	Sparse        *AccessorSparse `json:"sparse,omitempty"`
	Name          *string         `json:"name,omitempty"`
	Extensions    *SpecExtensions `json:"extensions,omitempty"`
	Extras        *Extras         `json:"extras,omitempty"`
}

func (s *SpecAccessor) SpecExtension() *SpecExtensions {
	return s.Extensions
}
func (s *SpecAccessor) Scheme() string {
	return SCHEME_ACCESSOR
}
func (s *SpecAccessor) Syntax(strictness Strictness, root Specifier, parent Specifier) error {
	switch strictness {
	case LEVEL3:
		if s.ByteOffset != nil && *s.ByteOffset < 0 {
			return errors.Errorf("Accessor.ByteOffset minimum(0)")
		}
		fallthrough
	case LEVEL2:
		if s.Max != nil && (1 > len(s.Max) || len(s.Max) > 16) {
			return errors.Errorf("Accessor.Max rangeitem(1, 16)")
		}
		if s.Min != nil && (len(s.Min) < 1 || len(s.Min) > 16) {
			return errors.Errorf("Accessor.Min rangeitem(1, 16)")
		}
		fallthrough
	case LEVEL1:
		if s.ByteOffset != nil && s.BufferView == nil {
			return errors.Errorf("Accessor.ByteOffset dependency(bufferView)")
		}
		if s.Count == nil {
			return errors.Errorf("Accessor.Count required")
		}
		if s.ComponentType == nil {
			return errors.Errorf("Accessor.ComponentType required")
		}
		if s.Type == nil {
			return errors.Errorf("Accessor.AccessorType required")
		}
	}
	return nil
}
func (s *SpecAccessor) To(ctx *parserContext) interface{} {
	res := new(Accessor)
	if s.ByteOffset == nil {
		res.ByteOffset = 0
	} else {
		res.ByteOffset = *s.ByteOffset
	}
	res.Count = *s.Count
	if s.Normalized == nil {
		res.Normalized = false
	} else {
		res.Normalized = *s.Normalized
	}
	res.ComponentType = *s.ComponentType
	res.Type = *s.Type
	res.Max = s.Max
	res.Min = s.Min
	// s.Sparse
	if s.Name == nil {
		res.Name = ""
	} else {
		res.Name = *s.Name
	}
	res.Extras = s.Extras
	return res
}
func (s *SpecAccessor) Link(Root *GLTF, parent interface{}, dst interface{}) error {
	if s.BufferView != nil {
		if !inRange(*s.BufferView, len(Root.BufferViews)) {
			return errors.Errorf("Accessor.BufferView linking fail, %s", *s.BufferView)
		}
		dst.(*Accessor).BufferView = Root.BufferViews[*s.BufferView]
	}
	return nil
}

func inKind(test reflect.Kind, set ...reflect.Kind) bool {
	for _, v := range set {
		if v == test {
			return true
		}
	}
	return false
}
