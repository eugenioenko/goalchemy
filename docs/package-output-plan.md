# Package-based output implementation plan

Implement the direction discussed in [issue #14](https://github.com/eugenioenko/goalchemy/issues/14): generated declarations belong to files grouped by their original Go package. Package-based output is the only layout. The existing native library import paths, exported APIs, readable/compact naming, runtime contracts and initialization lifecycle remain unchanged.

This is an implementation plan, not a claim that the new layout is available.

## Compiler boundaries

Keep one whole-program frontend, canonical type system, naming index and effect analysis. Preserve source-package identity and dependency order into emission. Functions, globals, closures and method wrappers need deterministic owners. Named and anonymous types can share canonical representations; choose and record one helper owner without duplicating runtime type identity.

Each backend routes declarations into package-owned files and emits the native linkage needed between them. Shared representations, runtime support and the existing central initializer may remain in dedicated shared files. Preserve source maps for each generated file and list every generated member in the build manifest. Repeated compilation must produce the same paths and contents.

Retain the current generated native package/module identities. This work does not introduce separately published dependencies for every source Go package. Go and Swift can share one native package/target across files; C# can use partial classes. TypeScript and Python need importable internal modules, Java needs native classes, Rust needs modules, and C needs independently compiled translation units with declarations in internal headers. Avoid concatenation, runtime source evaluation or textual inclusion as substitutes for native linkage.

The central initializer continues to initialize variables and call `init()` functions in Go dependency order. SDK-capable library operations retain fresh per-call source state. Initialization redesign, persistent clients and a naming redesign are outside this change.

## Implementation sequence

1. Preserve and verify ownership/dependency metadata; establish shared output-file and source-map handling.
2. Implement Go and TypeScript/Python output to prove native file linkage and mutable state across package boundaries.
3. Extend Java, C#, Rust, C and Swift. Update their runners and native package builds as each target lands.
4. Run focused regression checks during development, then the existing compiler CI matrix against the final PR head.
5. Pin the SDK to the reviewed compiler commit. Build and format all eight source distributions, commit them under root `dist/`, and verify independent consumers and existing real-KAS interoperability CI.

Root coordinates one implementation worker at a time, reviews results and makes signed Conventional Commits. Work occurs in isolated compiler and SDK worktrees. Preserve the existing distribution reference PR; open separate PRs for this work. No release publication or PR merge is part of this objective.

## Acceptance evidence

- A multi-package fixture proves package variables, dependency-ordered variable and `init()` effects, cross-package calls and mutation, closures, method/interface dispatch and Go value copying.
- Native executable and importing-library checks run across all eight targets. Readable and compact names retain their existing public interfaces.
- Layout checks verify that package-owned declarations actually reside in separate native files, that shared state/type identities are unique, and that no hidden combined source replaces them.
- Build manifests include all generated files. Per-file diagnostics map back to original source, and generation is deterministic.
- Existing compiler language/corpus, runtime, native capability, naming, float and importing-consumer suites pass without dropping required cases.
- All eight formatted SDK source distributions are committed. Their package imports and public APIs remain usable, with generated caches, binaries, credentials and local receipts excluded.
- The SDK's existing offline and eight-target focused real-KAS jobs pass, retaining its OpenTDF Go/Web oracles and required profile coverage.
- Both PRs have green CI at their final heads. Any CI synthetic merge tree is checked against the reviewed source tree before claiming equivalent evidence.

Document explicit constructors/setup as preferred for substantial initialization, including setup hidden in package-variable initializers. Small deterministic `init()` functions remain supported; do not silently discard initialization.

## Go implementation acceptance, 2026-10-08

Go now emits directly into package-owned native files in the existing generated
Go package. Named types, globals and functions retain their source owners;
initialization/API code remains in `main.go`, with canonical registration and
shared external support in `shared.go`. Source-package dependency metadata and
artifact paths/maps survive into the manifest. Receiver-wrapper ownership was
added without changing IDs or names.

Focused checks passed in readable and compact naming: the multi-package source
oracle, unique declaration ownership, same-basename package paths, deterministic
output, manifest completeness and physical source maps; importing libraries with
race checks, overlapping calls, fresh initialization and retained results/errors;
existing interface, embedded-method, closure, cooperative-frame and global-init
fixtures; and naming/CLI option checks. The combined fixture exposed a Go
interface-signature bug when an unrelated function value with the same signature
suspended. Interface declarations now follow the method identity's suspension
classification, and the unchanged regression passes.

Other seven backends remain pending. Before final acceptance, the shared driver
must remove obsolete compiler-managed files when reusing an output directory,
while preserving unlisted caller files. Final compiler CI, eight-target SDK
interoperability and committed formatted distributions are still required.
