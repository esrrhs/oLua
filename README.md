# oLua
[<img src="https://img.shields.io/github/license/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/languages/top/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/actions/workflow/status/esrrhs/oLua/go.yml?branch=master">](https://github.com/esrrhs/oLua/actions)

[English](./README.md) | [中文](./README_CN.md)

A practical Lua optimizer that safely speeds up table access, table constructors, hot global/library lookups, loop-invariant table paths, and constant folding / dead-code elimination.

## Optimizations
- [x] Optimize Lua table access (read/write analysis, safe and reliable)
- [x] Optimize Lua table constructors
- [x] Localize globals / library functions (`-opt_localize_globals`)
- [x] Loop-invariant code motion / LICM (`-opt_licm`)
- [x] Constant folding / dead code elimination (`-opt_const_fold`)


## Safety-first optimization policy

The passes are opt-in. Runtime differential tests now take precedence over old
textual golden outputs. Unknown effects mean **skip**, not "probably pure".
Only function bodies are optimized; top-level chunks are not rewritten.

### Table access and LICM

`-opt_table_access` and `-opt_licm` require a closed private-table proof:

- Tables are fresh, literal-only locals in the function's outer block.
- All accessed fields and intermediate tables are known to exist.
- No calls, captured variables, nested closures, generic iterators, table writes,
  unknown inputs, or shadowing are accepted. Tables cannot escape to calls/stores
  before caching; returning them at function exit is allowed.
- Cache declarations dominate their uses. Dotted paths must be editable as code
  tokens; inline/multiline-string cases are conservatively skipped.
- This makes speculative reads safe even for a zero-trip loop: there is no nil
  parent or `__index` to evaluate. Parameter/global tables are **not** hoisted.

```lua
local function sum()
    local cfg = {weight = {x=2, y=3}}
    local total = 0
    for i=1,100 do
        total = total + i*cfg.weight.x
        total = total + i*cfg.weight.y
    end
    return total
end
```

Here LICM may cache `cfg.weight` before the loop. Arbitrary `self.x.y` game state
is not proven safe. A stable spelling is not proof of stable table semantics.
`print`, `tostring`, `pairs`, and `math.*` can be rebound or invoke metamethods.
The legacy `-opt_table_access_pure_funcs` and `-opt_table_access_global` flags are
retained for CLI compatibility; they do **not** bypass the new proof.

Both read thresholds default to 2 and are clamped to at least 2:
`-opt_table_access_threshold`, `-opt_licm_threshold`.

### Table constructors

`-opt_table_constructor` merges only adjacent, standalone, literal-only writes
into a fresh local constructor, e.g. `local a={}; a.x=2` on separate lines.
No global/field targets, RHS calls/variable reads, intervening statements,
numeric/sequence keys, or multiline source edits are merged. Lua strings are escaped
as Lua bytes, not inserted as unescaped AST text.

### Globals / library localization

`-opt_localize_globals` alone now conservatively leaves global lookups unchanged.
Lua's `_ENV` and library tables can be rebound or implement effectful `__index`;
the optimizer currently cannot prove their stability across calls.

For code whose owner can guarantee a **plain, effect-free, stable environment,
stable libraries/functions, and no indirect rebinding**, the explicit unsafe
contract restores whitelist localization:

```bash
./oLua -input in.lua -output out.lua -opt_localize_globals \
  -unsafe_assume_static_globals
```

This opt-in is **not a semantic guarantee**. Its threshold is
`-opt_localize_globals_threshold` (default 2). Runtime equivalence/matrix tests
use safe defaults; a separate test covers the stable-environment contract.

### Constant folding / DCE

`-opt_const_fold` folds literal arithmetic/comparison/boolean expressions and
eliminates `if true`/`if false` branches while retaining `do/end` lexical scopes
(including branch-final `return`). Source edits use code-token boundaries and
parenthesize folded values; strings/comments are never matched as expressions.

Target: Lua 5.3/5.4 with 64-bit integers and IEEE double numbers. Integer `//`, `%`,
comparisons, and wrapping arithmetic stay integer-exact, including large values.
Mixed comparisons outside exact float range, escaped string constants, floating
division/modulo/floor division/power, division by zero, and nonfinite results are
left to Lua. At most one expression per original marked source line is folded.
`goto`/`break`/labels block DCE; ambiguous same-line `if` locations are skipped.
The parser does not support all Lua 5.4 syntax (e.g. local attributes).

Pipeline: const fold/DCE → constructor → localization → LICM → table access.
All flags default off. Reparse iterations are bounded to 1,000, with an error on
nonconvergence rather than an infinite optimizer run.

## Usage

Build:
```bash
go mod tidy
go build
```

Optimize table access for one file:
```bash
./oLua -input input.lua -output output.lua -opt_table_access
```

Optimize table constructors for one file:
```bash
./oLua -input input.lua -output output.lua -opt_table_constructor
```

Localize globals / library functions:
```bash
./oLua -input input.lua -output output.lua -opt_localize_globals
```

Loop-invariant code motion:
```bash
./oLua -input input.lua -output output.lua -opt_licm
```

Constant folding / DCE:
```bash
./oLua -input input.lua -output output.lua -opt_const_fold
```

Enable all and batch-process a directory (in-place):
```bash
./oLua -inputpath input_dir -opt_table_access -opt_table_constructor -opt_localize_globals -opt_licm -opt_const_fold
```

In batch mode, a parse failure on one file does not abort the run; that file is skipped and others continue.

## Validation and performance

Lua 5.3 or 5.4 is **required**, not silently skipped by `go test`. Set `OLUA_LUA`
to an executable, or a colon-separated list to require multiple runtimes.

```bash
go test -count=1 ./...
go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go vet ./...
go tool cover -func=coverage.out
python3 scripts/benchmark_lua.py --lua lua --output lua-benchmark.json
```

See [validation, limitations, and measured results](docs/VALIDATION.md). The
Python benchmark runs real Lua (not Go optimizer timing), checks workload
checksums, alternates paired timing order, reports all samples/variability and
original/optimized source bytes. Small kernel wins are not application speedup
promises. CI requires both Lua 5.3/5.4, race tests, coverage, and a benchmark smoke.

## Also see
[lua family bucket](https://github.com/esrrhs/lua-family-bucket)
