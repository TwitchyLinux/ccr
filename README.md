# Core Contracts

(ccr = Core Contracts Resolver, the tool that works with core contracts).

## What is it?

CoreContracts is tooling to assemble system components into a Linux system image. System
components are specified in forms that capture their inter-dependencies, allowing the
system to be assembled smartly and heavily validated.

## Whats wrong with existing ways to build a Linux system?

In my experience, there are two main ways to build up a Linux system:

 * Build / obtain some stuff, copy-pasta them into the right places, pray it works
 * Use a package manager, which does the above, but packages embody nuance or logic
   so they work & play nicely, most of the time

The problem with both of these approaches is the details about _how_ dependencies
are embodied. Scripts tend to implicitly encode the nuance of inter-dependencies and the assumptions
of the maintainer, rather than be obvious to an observer trying to change or fix something. Due to
the complexity of a modern system, any change is fraught with the danger of breaking some unknown
dependencies.

Package managers are better in this respect, but have some of the same issues. Decisions
around what should be contained within one package or another, which logic should exist in
pre/post/configure, and issues with under-specified relationships with other components
(either due to the limitations of specifying relationships only on other packages, or accidental
omission) all compound to embed tacit nuance in the layout of packages. Again, these hidden
details frustrate the process of detecting and correcting issues, and reduce our confidence that
an assembled Linux system will actually work.

With that in mind, CoreContracts tries to:

 * Exhaustively or near-exhaustively express the inter-dependencies between all the different
   components making up a Linux system.
 * Validate that the specified inter-dependencies hold true.
 * Group like system components together, enabling automatic validation of classes of system
   components (eg: All system libraries could be validated to be well-formed ELF for the intended
   architecture, present in the runtime-linker cache, and have their runtime library requirements
   satisfied).
