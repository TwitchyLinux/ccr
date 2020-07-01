// Package info implements runtime gathering of information, to be
// used by later generators or checkers.
package info

// ELFPopulator reads information from the ELF header of a binary.
var ELFPopulator = &elfPopulator{}

// StatPopulator computes the path and os.Stat of a path.
var StatPopulator = &statPopulator{}
