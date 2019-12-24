package ccr

import (
	"fmt"

	"github.com/twitchylinux/ccr/vts"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

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
			return ErrNotExists(t.Path)
		}
	}

	if err := u.generateTarget(generationState{
		basePath:      basePath,
		conf:          &conf,
		runnerEnv:     &opts,
		haveGenerated: make(targetSet, 4096),
		rootTarget:    target,
	}, target); err != nil {
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
	// rootTarget represents the topmost target for which dependencies and inputs
	// are currently being resolved. This will be the height target in the tree
	// which has inputs defined, or the root node.
	rootTarget vts.Target
}

func (u *Universe) generateTarget(s generationState, t vts.Target) error {
	// If target is a decendant of a set of inputs, we check to make sure
	// it hasn't already been seen, which would symbolize a circular dependency.
	if s.isGeneratingInputs {
		if _, alreadyDep := s.inputDep[t]; alreadyDep {
			if gt, ok := s.rootTarget.(vts.GlobalTarget); ok {
				return fmt.Errorf("circular dependency at %q from %q", t.(vts.GlobalTarget).GlobalPath(), gt.GlobalPath())
			}
			return fmt.Errorf("circular dependency at %q from %v", t.(vts.GlobalTarget).GlobalPath(), s.rootTarget)
		}
		s.inputDep[t] = struct{}{}
	}
	if _, alreadyGenerated := s.haveGenerated[t]; alreadyGenerated {
		return nil
	}

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
				return err
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
				return err
			}
		}
	}

	// As inputs have already been evaluated, the only remaining source of nested
	// dependencies is deps. We process these last.
	if deps, hasDeps := t.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.generateTarget(s, dep.Target); err != nil {
				return err
			}
		}
	}

	// Lastly, we generate the current target.
	if st, hasSrc := t.(vts.SourcedTarget); hasSrc {
		if src := st.Src(); src != nil {
			// Recurse to make sure all dependencies are resolved.
			if err := u.generateTarget(s, src.Target); err != nil {
				return err
			}
			if err := u.generateResourceUsingSource(s, st.(*vts.Resource), src.Target); err != nil {
				return err
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
