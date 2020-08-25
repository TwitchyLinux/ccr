package ccr

import (
	"errors"
	"fmt"
	"sort"

	"github.com/twitchylinux/ccr/log"
	"github.com/twitchylinux/ccr/vts"
)

const maxDepOrderIterations = 80

type collectionState struct {
	allDeps map[vts.Target][]vts.Target
}

// TargetsDependencyOrder returns a sets of targets of the given type, where
// the order represents dependency order, and targets in the same set may
// be generated simultaneously.
func (u *Universe) TargetsDependencyOrder(conf GenerateConfig, t vts.TargetRef, basePath string, tt vts.TargetType) ([][]vts.Target, error) {
	if !u.resolved {
		return nil, ErrNotBuilt
	}

	var target vts.Target
	if t.Target == nil {
		var ok bool
		target, ok = u.fqTargets[t.Path]
		if !ok {
			u.logger.Error(log.MsgBadFind, ErrNotExists(t.Path))
			return nil, ErrNotExists(t.Path)
		}
	}

	runnerEnv := u.MakeEnv(basePath)
	cs := collectionState{
		allDeps: make(map[vts.Target][]vts.Target, 64),
	}
	if err := u.collectDeps(generationState{
		basePath:               basePath,
		conf:                   &conf,
		runnerEnv:              runnerEnv,
		haveGenerated:          make(targetSet, 4096),
		targetChain:            make([]vts.Target, 0, 64),
		rootTarget:             target,
		completedToolchainDeps: make(targetSet, 32),
	}, target, nil, &cs); err != nil {
		u.logger.Error(log.MsgFailedPrecondition, err)
		return nil, err
	}

	pending, emitted := make(map[vts.Target]struct{}, len(cs.allDeps)/2), make(map[vts.Target]struct{}, len(cs.allDeps))
	for k, _ := range cs.allDeps {
		if k.TargetType() == tt {
			pending[k] = struct{}{}
		}
	}

	var out [][]vts.Target

	for i := 0; len(pending) > 0; i++ {
		curSet := make([]vts.Target, 0, 6)
		if i > maxDepOrderIterations {
			return nil, errors.New("max resolution iterations were reached")
		}

		for k, _ := range pending {
			// fmt.Printf("Pending[%02d]: %v\n", i, k)
			pendingDeps := numDepsOfType(cs.allDeps[k], emitted, tt)
			// fmt.Printf("  pending = %d (%v)\n", pendingDeps, cs.allDeps[k])
			if pendingDeps == 0 {
				curSet = append(curSet, k)
			}
		}

		sort.Slice(curSet, func(i int, j int) bool {
			oi, ok := curSet[i].(fmt.Stringer)
			if !ok {
				return false
			}
			oj, ok := curSet[j].(fmt.Stringer)
			if !ok {
				return false
			}
			return oi.String() < oj.String()
		})
		for _, s := range curSet {
			delete(pending, s)
			emitted[s] = struct{}{}
		}
		out = append(out, curSet)
	}

	return out, nil
}

func numDepsOfType(deps []vts.Target, ignore map[vts.Target]struct{}, tt vts.TargetType) int {
	out := 0
	for _, d := range deps {
		if _, ignore := ignore[d]; ignore {
			continue
		}
		if d.TargetType() == tt {
			out++
		}
	}
	return out
}

func (u *Universe) collectDeps(s generationState, t, parent vts.Target, cs *collectionState) error {
	// If target is a decendant of a set of inputs, we check to make sure
	// it hasn't already been seen, which would symbolize a circular dependency.
	if s.isGeneratingInputs {
		for _, c := range s.targetChain {
			if c == t {
				return s.makeCircularDepErr(t)
			}
		}
	}
	if cs.allDeps[t] == nil {
		cs.allDeps[t] = make([]vts.Target, 0, 2)
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
		for _, inp := range inputs.NeedInputs() {
			subState := generationState{
				isGeneratingInputs:     true,
				haveGenerated:          make(targetSet, 128),
				basePath:               s.basePath,
				conf:                   s.conf,
				runnerEnv:              s.runnerEnv,
				rootTarget:             t,
				completedToolchainDeps: s.completedToolchainDeps,
				targetChain:            append(make([]vts.Target, 0, 6), t),
			}
			if err := u.collectDeps(subState, inp.Target, t, cs); err != nil {
				return vts.WrapWithTarget(err, inp.Target)
			}
			cs.allDeps[t] = deDupe(append(cs.allDeps[t], append(cs.allDeps[inp.Target], inp.Target)...))
		}
	}

	// Specifying a class target as a dependency or input actually means all
	// instances of that class are a dependency.
	if t.IsClassTarget() {
		for _, classInstance := range u.classedTargets[t] {
			if err := u.collectDeps(s, classInstance, t, cs); err != nil {
				return vts.WrapWithTarget(err, classInstance)
			}
		}
	}

	// As inputs have already been evaluated, the only remaining source of nested
	// dependencies is deps. We process these last.
	if deps, hasDeps := t.(vts.DepTarget); hasDeps {
		for _, dep := range deps.Dependencies() {
			if err := u.collectDeps(s, dep.Target, t, cs); err != nil {
				return vts.WrapWithTarget(err, dep.Target)
			}
		}
	}

	if st, hasSrc := t.(vts.SourcedTarget); hasSrc {
		if src := st.Src(); src != nil {
			if err := u.collectDeps(s, src.Target, t, cs); err != nil {
				return vts.WrapWithTarget(err, src.Target)
			}
		}
	}

	s.haveGenerated[t] = struct{}{}
	if parent == nil {
		cs.allDeps[t] = append(cs.allDeps[t], t)
	} else {
		cs.allDeps[parent] = deDupe(append(cs.allDeps[parent], append(cs.allDeps[t], t)...))
	}
	return nil
}

func deDupe(set []vts.Target) []vts.Target {
	deDupe := make(map[vts.Target]struct{}, len(set))
	for _, st := range set {
		deDupe[st] = struct{}{}
	}
	out := make([]vts.Target, 0, len(deDupe))
	for st, _ := range deDupe {
		out = append(out, st)
	}
	return out
}
