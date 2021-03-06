package gltf2

import (
	"encoding/json"
	"fmt"
	"github.com/iamGreedy/glog"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type parser struct {
	dst    *GLTF
	src    *SpecGLTF
	parsed bool
	// extension support
	exts []ExtensionType
	//
	cause error
	err   error
	//
	strictness Strictness
	rd         io.Reader
	dir        http.Dir
	// [progress]
	// + logging task
	logger *glog.Glogger
	//
	pres  []PreTask
	posts []PostTask
}

func (s *parser) Reader(reader io.Reader) *parser {
	if reader == nil {
		s.setCauseError(ErrorParserOption, errors.Errorf("nil reader"))
		return s
	}
	s.rd = reader
	if f, ok := s.rd.(*os.File); ok {
		return s.Directory(filepath.Dir(f.Name()))
	}
	return s
}
func (s *parser) Strictness(strictness Strictness) *parser {
	if strictness < LEVEL0 {
		strictness = LEVEL0
	}
	if strictness > LEVEL3 {
		strictness = LEVEL3
	}
	s.strictness = strictness
	return s
}
func (s *parser) Tasks(tasks ...Task) *parser {
	for _, task := range tasks {
		if prp, ok := task.(PreTask); ok {
			s.pres = append(s.pres, prp)
		}
		if pop, ok := task.(PostTask); ok {
			s.posts = append(s.posts, pop)
		}
	}
	return s
}
func (s *parser) Extensions(exts ...ExtensionType) *parser {
	s.exts = append(s.exts, exts...)
	return s
}
func (s *parser) Directory(path string) *parser {
	fi, err := os.Stat(path)
	if err != nil {
		s.setCauseError(ErrorParserOption, err)
	}
	if !fi.IsDir() {
		s.setCauseError(ErrorParserOption, errors.Errorf("Not directory '%s'", path))
	} else {
		s.dir = http.Dir(path)
	}
	return s
}
func (s *parser) Logger(dst io.Writer) *parser {
	if dst != nil {
		s.logger = glog.New(log.New(dst, "[ glTF 2.0 ] ", log.LstdFlags), "    ")
	} else {
		s.setCauseError(ErrorParserOption, errors.Errorf("Parser().Logger(<not nillable>)"))
	}
	return s
}
func (s *parser) GetLogger() *glog.Glogger {
	return s.logger
}

//

func (s *parser) Error() error {
	if s.src == nil {
		return errors.WithMessage(ErrorParser, "Closed Parse")
	}
	if s.cause != nil {
		s.logger.Println(s.err)
		return errors.WithMessage(s.cause, s.err.Error())
	}
	if s.err != nil {
		s.logger.Println(s.err)
		return errors.WithMessage(ErrorParser, s.err.Error())
	}
	return nil
}
func (s *parser) setError(err error) {
	s.cause = nil
	s.err = err
}
func (s *parser) setCauseError(cause error, err error) {
	s.cause = cause
	s.err = err
}
func (s *parser) Parsed() bool {
	return s.parsed
}
func (s *parser) Parse() (*GLTF, error) {
	if err := s.Error(); err != nil {
		return nil, err
	}
	if s.parsed {
		return s.dst, nil
	}
	//====================================================//
	// constraint
	if s.rd == nil {
		s.setCauseError(ErrorParser, errors.Errorf("No reader"))
		return nil, s.Error()
	}
	//
	ctx := &parserContext{
		ref: s,
	}
	//====================================================//
	// json parse here
	dec := json.NewDecoder(s.rd)
	s.logger.Println("json decode start...")
	if err := dec.Decode(s.src); err != nil {
		s.setCauseError(ErrorJSON, err)
		return nil, s.Error()
	}
	s.logger.Println("json decode complete")
	//====================================================//
	// extension used
	for _, v := range s.src.ExtensionsRequired {
		if !inExtension(v, s.exts) {
			s.setCauseError(ErrorExtension, errors.Errorf("ExtensionKey : '%s' not support", v))
			return nil, s.Error()
		}
	}
	//====================================================//
	// extension parse here
	s.logger.Println("extension json decode start...")
	if err := recurPostJson(s.src, s.exts); err != nil {
		s.setCauseError(ErrorExtension, err)
		return nil, s.Error()
	}
	s.logger.Println("extension json decode complete")
	//====================================================//
	// pre Task
	if s.pres != nil {
		s.logger.Println("PreTasks start")
		for i, pre := range s.pres {
			s.logger.Printf("PreTask(%d/%d) '%s'\n", i+1, len(s.pres), pre.TaskName())
			pre.PreLoad(ctx, s.src, s.logger.Indent())
		}
		s.logger.Println("PreTasks complete")
	}
	//====================================================//
	// Syntax check
	s.logger.Println("syntax check")
	s.logger.Printf("syntax strictness : %v \n", s.strictness)
	if err := recurSyntax(s.src, nil, s.src, s.strictness); err != nil {
		s.setCauseError(ErrorGLTFSpec, err)
		return nil, s.Error()
	}
	s.logger.Println("syntax valid")
	//====================================================//
	// Convert
	s.dst = s.src.To(nil).(*GLTF)
	s.logger.Println("structure setup")
	recurTo(s.dst, s.src, ctx)
	s.logger.Println("structure setup complete")
	//====================================================//
	// Link
	s.logger.Println("glTFid reference linking start")
	if err := recurLink(s.dst, nil, s.dst, s.src); err != nil {
		s.setCauseError(ErrorGLTFLink, err)
		return nil, s.Error()
	}
	s.logger.Println("glTFid reference linked")
	//====================================================//
	if s.posts != nil {
		s.logger.Println("PostTask start")
		// post Task
		for i, post := range s.posts {
			s.logger.Printf("PostTask(%d/%d) '%s'\n", i+1, len(s.posts), post.TaskName())
			if err := post.PostLoad(ctx, s.dst, s.logger.Indent()); err != nil {
				s.setCauseError(ErrorTask, errors.WithMessage(err, fmt.Sprintf("Task name : %s", post.TaskName())))
				return nil, s.Error()
			}
		}
		s.logger.Println("PostTask complete")
	}
	s.logger.Println("complete.")
	return s.dst, nil
}
func (s *parser) Close() error {
	s.src = nil
	return nil
}

func Parser() *parser {
	res := &parser{
		src: new(SpecGLTF),

		strictness: LEVEL1,
		dir:        http.Dir("."),
	}
	return res
}

func recurPostJson(target Specifier, exts []ExtensionType) error {
	if target == nil {
		return nil
	}
	//// extension
	if g, ok := target.(ExtensionSpecifier); ok {
		if ext := g.SpecExtension(); ext != nil {
			var delList []string
			for k, v := range *ext {
				if exk := extFindByName(k, exts...); exk != nil {
					var err error
					v.data, err = exk.Constructor(v.src)
					if err != nil {
						return err
					}
				} else {
					delList = append(delList, k)
				}
			}
			for _, v := range delList {
				delete(*ext, v)
			}
		}
	}
	//
	if tc, ok := target.(Parents); ok {
		for i := 0; i < tc.LenChild(); i++ {
			if err := recurPostJson(tc.GetChild(i), exts); err != nil {
				return err
			}
		}
	}
	return nil
}
func recurSyntax(root, parent, target Specifier, strictness Strictness) error {
	if target == nil {
		return nil
	}
	if err := target.Syntax(strictness, root, parent); err != nil {
		return err
	}
	// extension
	if g, ok := target.(ExtensionSpecifier); ok {
		if ext := g.SpecExtension(); ext != nil {
			for _, v := range *ext {
				if err := v.data.Syntax(strictness, root, parent); err != nil {
					return err
				}
			}
		}
	}
	//
	if tc, ok := target.(Parents); ok {
		for i := 0; i < tc.LenChild(); i++ {
			if err := recurSyntax(root, target, tc.GetChild(i), strictness); err != nil {
				return err
			}
		}
	}
	return nil
}
func recurTo(data interface{}, target Specifier, ctx *parserContext) {
	if g, ok := target.(ExtensionSpecifier); ok {
		var dataext = make(Extensions)
		if specext := g.SpecExtension(); specext != nil {
			for k, v := range *specext {
				extk := extFindByName(k, ctx.ref.exts...)
				extData := v.data.To(ctx)
				recurTo(extData, v.data, ctx)
				dataext[extk.ExtensionName()] = extData.(ExtensionType)
			}
		}
		data.(ExtensionStructure).SetExtension(&dataext)
	}
	if tc, ok := target.(Parents); ok {
		l := tc.LenChild()

		for i := 0; i < l; i+= 1 {
			child := tc.GetChild(i)
			childData := child.To(ctx)
			recurTo(childData, child, ctx)
			tc.SetChild(i, data, childData)
		}
	}
}
func recurLink(root *GLTF, parent, data interface{}, target Specifier) error {
	if tl, ok := target.(Linker); ok {
		if err := tl.Link(root, parent, data); err != nil {
			return err
		}
	}
	if g, ok := target.(ExtensionSpecifier); ok {
		specext := g.SpecExtension()
		dataext := data.(ExtensionStructure).GetExtension()
		if dataext != nil {
			for k, v := range *dataext {
				if err := recurLink(root, data, v, (*specext)[k].data); err != nil {
					return err
				}
			}
		}
	}
	if tc, ok := target.(Parents); ok {
		for i := 0; i < tc.LenChild(); i++ {
			if err := recurLink(root, data, tc.ImpleGetChild(i, data), tc.GetChild(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

type parserContext struct {
	ref *parser
}

func (s *parserContext) Strictness() Strictness {
	return s.ref.strictness
}
func (s *parserContext) Directory() string {
	return string(s.ref.dir)
}
func (s *parserContext) Specification() *SpecGLTF {
	return s.ref.src
}
func (s *parserContext) GLTF() *GLTF {
	return s.ref.dst
}
