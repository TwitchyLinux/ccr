package ccr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

// CircularDependencyError is returned if a circular dependency creates
// a situation where generation is not possible.
type CircularDependencyError struct {
	msg  string
	Deps []vts.Target
}

func (c CircularDependencyError) Error() string {
	return c.msg
}

// GenerateConfig describes parameters to use when generating against
// a universe.
type GenerateConfig struct{}

// Generate applies the tree of rules in target to basePath, creating a
// system based on those rules.
func (u *Universe) Generate(conf GenerateConfig, t vts.TargetRef, basePath string) error {
	if !u.resolved {
		return ErrNotBuilt
	}
	opts := vts.RunnerEnv{
		Dir: basePath,
		FS:  osfs.New(basePath),
	}
	var target vts.Target

	if t.Target == nil {
		var ok bool
		target, ok = u.fqTargets[t.Path]
		if !ok {
			u.logger.Error(MsgBadFind, ErrNotExists(t.Path))
			return ErrNotExists(t.Path)
		}
	}

	if err := u.generateTarget(generationState{
		basePath:      basePath,
		conf:          &conf,
		runnerEnv:     &opts,
		haveGenerated: make(targetSet, 4096),
		targetChain:   make([]vts.Target, 0, 64),
		rootTarget:    target,
	}, target); err != nil {
		u.logger.Error(MsgFailedPrecondition, err)
		return err
	}

	checked := make(targetSet, 4096)
	return u.checkTarget(target, &opts, checked)
}

type generationState struct {
	// basePath refers to the directory generated artifacts should reside.
	basePath string
	// conf encapsulates the configuration for generation.
	conf *GenerateConfig
	// runnerEnv is how generators manipulate the output directory.
	runnerEnv *vts.RunnerEnv
	// isGeneratingInputs is true when the target is being evaluated as a
	// decendant of a generator's inputs. When true, new dependencies or inputs
	// must be checked against inputDep to determine if a circular dependency
	// exists.
	isGeneratingInputs bool
	// haveGenerated keeps track of targets which have already
	// been generated.
	haveGenerated targetSet
	// inputDep exhaustively enumerates targets which are part of a generator
	// input which is currently being examined.
	inputDep targetSet
	// targetChain enumerates targets from the root to the current target.
	targetChain []vts.Target
	// rootTarget represents the topmost target for which dependencies and inputs
	// are currently being resolved. This will be the height target in the tree
	// which has inputs defined, or the root node.
	rootTarget vts.Target
}

func (s generationState) makeCircularDepErr(t vts.Target) error {
	rootIdx := 0
	for i := 0; i < len(s.targetChain); i++ {
		if s.targetChain[i] == s.rootTarget {
			rootIdx = i
			break
		}
	}

	depChain := s.targetChain[rootIdx:]
	msg := "circular dependency: "
	for _, t := range depChain {
		if gt, ok := t.(vts.GlobalTarget); ok {
			msg += gt.GlobalPath() + " -> "
		} else {
			msg += fmt.Sprintf("anon<%s> -> ", t.TargetType())
		}
	}
	if gt, ok := t.(vts.GlobalTarget); ok {
		msg += gt.GlobalPath()
	} else {
		msg += fmt.Sprintf("anon<%s>", t.TargetType())
	}

	return CircularDependencyError{
		msg:  msg,
		Deps: depChain,
	}
}

func (u *Universe) generateTarget(s generationState, t vts.Target) error {
	// If target is a decendant of a set of inputs, we check to make sure
	// it hasn't already been seen, which would symbolize a circular dependency.
	if s.isGeneratingInputs {
		if _, alreadyDep := s.inputDep[t]; alreadyDep {
			return s.makeCircularDepErr(t)
		}
		s.inputDep[t] = struct{}{}
	}
	if _, alreadyGenerated := s.haveGenerated[t]; alreadyGenerated {
		return nil
	}
	// Update targetChain.
	s.targetChain = append(s.targetChain, t)
	defer func() {
		s.targetChain = s.targetChain[:len(s.targetChain)-1]
	}()

	// Inputs cannot have circular dependencies, so we evaluate them first and
	// in a different mode to detect the circular dependencies.
	if inputs, hasInputs := t.(vts.InputTarget); hasInputs {
		subState := generationState{
			isGeneratingInputs: true,
			haveGenerated:      s.haveGenerated,
			inputDep:           make(targetSet, 128),
			basePath:           s.basePath,
			conf:               s.conf,
			runnerEnv:          s.runnerEnv,
			rootTarget:         t,
		}
		subState.inputDep[t] = struct{}{}
		for _, inp := range inputs.NeedInputs() {
			if err := u.generateTarget(subState, inp.Target); err != nil {
				return vts.WrapWithTarget(err, inp.Target)
			}
		}
	}

	// Specifying a class target as a dependency or input actually means all
	// instances of that class are a dependency.
	if t.IsClassTarget() {
		// for c, i := range u.classedTargets {
		// 	fmt.Printf("class = %+v\n", c)
		// 	for _, i := range i {
		// 		fmt.Printf("  %+v\n", i)
		// 	}
		// }
		// fmt.Println(t, u.classedTargets[t.(vts.Target)])
		for _, classInstance := range u.classedTargets[t] {
			if err := u.generateTarget(s, classInstance); err != nil {
				return vts.WrapWithTarget(err, classInstance)
			}
		}
	}

	// As inputs have already been evaluated, the only remaining source of nested
	// dependencies is deps. We process these last.
	if deps, hasDeps := t.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.generateTarget(s, dep.Target); err != nil {
				return vts.WrapWithTarget(err, dep.Target)
			}
		}
	}

	// Lastly, we generate the current target.
	if st, hasSrc := t.(vts.SourcedTarget); hasSrc {
		if src := st.Src(); src != nil {
			// Recurse to make sure all dependencies are resolved.
			if err := u.generateTarget(s, src.Target); err != nil {
				return vts.WrapWithTarget(err, src.Target)
			}
			if err := u.generateResourceUsingSource(s, st.(*vts.Resource), src.Target); err != nil {
				return vts.WrapWithTarget(err, src.Target)
			}
		}
	}

	s.haveGenerated[t] = struct{}{}
	return nil
}

func (u *Universe) generateResourceUsingSource(s generationState, resource *vts.Resource, source vts.Target) error {
	info := vts.InputSet{
		Resource: resource,
		Directs:  make([]vts.Target, len(resource.Deps)),
	}
	for i := range resource.Deps {
		info.Directs[i] = resource.Deps[i].Target
	}

	switch src := source.(type) {
	case *vts.Puesdo:
		if src.Kind != vts.FileRef {
			return fmt.Errorf("cannot generate using puesdo source %v", src.Kind)
		}
		return u.generateFileSource(s, resource, src)

	case *vts.Generator:
		info.ClassedResources = map[*vts.ResourceClass][]*vts.Resource{}
		for i, inp := range src.Inputs {
			switch input := inp.Target.(type) {
			case *vts.Resource, *vts.Component:
				info.Directs = append(info.Directs, input)
			case *vts.ResourceClass:
				resList := make([]*vts.Resource, len(u.classedTargets[inp.Target]))
				for i, res := range u.classedTargets[inp.Target] {
					resList[i] = res.(*vts.Resource)
				}
				info.ClassedResources[input] = resList
			default:
				return fmt.Errorf("input[%d] references unsupported target type %T", i, inp.Target)
			}
		}
		return src.Run(resource, &info, s.runnerEnv)
	}

	return fmt.Errorf("cannot generate using source %T for resource %v", source, resource)
}

func fileSrcInfo(resource *vts.Resource, src *vts.Puesdo) (string, string, os.FileMode, error) {
	outFilePath, err := determinePath(resource)
	if err != nil {
		return "", "", 0, err
	}
	srcFilePath := filepath.Join(filepath.Dir(src.ContractPath), src.Path)
	mode, err := determineMode(resource)
	switch {
	case err == errNoAttr:
		st, err := os.Stat(srcFilePath)
		if err != nil {
			return "", "", 0, vts.WrapWithPath(err, srcFilePath)
		}
		mode = st.Mode() & os.ModePerm
	case err != nil:
		return "", "", 0, err
	}
	return srcFilePath, outFilePath, mode, nil
}

func (u *Universe) generateFileSource(s generationState, resource *vts.Resource, src *vts.Puesdo) error {
	srcPath, outPath, mode, err := fileSrcInfo(resource, src)
	if err != nil {
		return err
	}

	r, err := os.OpenFile(srcPath, os.O_RDONLY, 0644)
	if err != nil {
		return vts.WrapWithPath(err, srcPath)
	}
	defer r.Close()

	if dir, _ := filepath.Split(outPath); dir != "" && dir != "/" {
		if err := s.runnerEnv.FS.MkdirAll(dir, 0755); err != nil {
			return vts.WrapWithPath(err, dir)
		}
	}

	w, err := s.runnerEnv.FS.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	defer w.Close()
	if _, err := io.Copy(w, r); err != nil {
		return vts.WrapWithPath(err, outPath)
	}
	return nil
}
