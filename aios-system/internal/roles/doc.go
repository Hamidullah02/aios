// Package roles is the agent role registry for
// aios.
//
// The package provides three things:
//
//   - Schema: the Role typed object that holds
//     the agent configuration (system prompt,
//     tool set, model class, output format,
//     temperature, max tokens, stop sequences).
//   - Registry: the in-memory role table with
//     get/set/list and a YAML loader. The
//     registry exposes the canonical default
//     roles (planner, coder, editor, explorer,
//     reviewer, verifier).
//   - Resolver: the lookup that returns the
//     configuration for a role name. The
//     resolver supports a fallback role and
//     raises UnknownRoleError when the name is
//     not in the registry and no fallback is
//     set.
//
// The package depends on aios-core (for the
// canonical JSON encoder and the small
// validation helpers) and the Go standard
// library plus gopkg.in/yaml.v3 (already a
// direct dep of the monorepo, no new deps).
// The package is depended on by aios-planner,
// aios-runtime, aios-watch, and aios-cli (per
// the doc set).
//
// Per the doc set the package also depends on
// aios-models. The implementation does not
// import aios-models: the model's class is
// stored as a string (cheap/mid/strong) and
// the model picker lives one layer up. The
// doc-set note about "model class" maps to
// the ModelClass field on the Role; the
// concrete model is resolved at call time by
// the model router (aios-models), not by the
// role package.
package roles
