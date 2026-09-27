# oLua
[<img src="https://img.shields.io/github/license/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/languages/top/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/actions/workflow/status/esrrhs/oLua/go.yml?branch=master">](https://github.com/esrrhs/oLua/actions)

[English](./README.md) | [中文](./README_CN.md)

A practical Lua optimizer that safely speeds up table access and table constructors using read/write analysis.

## Optimizations
- [x] Optimize Lua table access (read/write analysis, safe and reliable)
- [x] Optimize Lua table constructors

## TODO
- [ ] **Localize globals / library functions**: hoist `local print, format = print, string.format` at hot function entries to reduce `_ENV` lookups
- [ ] **Loop-invariant code motion (LICM)**: hoist expressions that do not depend on the loop iterator and have no writes along the path
- [ ] **Constant folding / dead code elimination**: `2+3*4` → `14`; drop `if false then ... end`; rewrite `if true then A else B end` → `A`

## Optimize Lua table access

### Basics

Every `a.b` in Lua triggers a table lookup. If the same table path is read multiple times in a scope, caching it into a local variable reduces lookups.

For example:
```lua
local x = a.b.c
local y = a.b.d
local z = a.b.e
```
becomes:
```lua
local a_b = a.b -- opt by oLua
local x = a_b.c
local y = a_b.d
local z = a_b.e
```

### Read/Write analysis

Unlike simple pattern matching, oLua uses **read/write analysis** to decide when a cache is still valid:

- **Writes invalidate caches**: `a.b = {}`, `a = ...` (parent write), `func1(a)` (the function may mutate fields of `a`), etc.
- **Reads after a write need a refreshed cache**

For example:
```lua
local x = a.b.c
local y = a.b.d
func1(a)         -- may mutate a, invalidating a.b
local z = a.b.e
local w = a.b.f
```
becomes:
```lua
local a_b = a.b -- opt by oLua
local x = a_b.c
local y = a_b.d
func1(a)
a_b = a.b -- opt by oLua (auto refresh)
local z = a_b.e
local w = a_b.f
```

### Invalidation details

- `func1(a.b)` passes the **value** of `a.b`; it cannot rebind field `b` on table `a`, so the **`a.b` cache stays valid**
- Inside `func1(a.b)`, the function may mutate fields of `a.b` (e.g. `arg.x = 1`), so **child caches like `a.b.x` are invalidated**
- `func1(a)` passes the whole table `a`, so `a.b = xxx` is possible and the **`a.b` cache is invalidated**
- In `a.b:method()`, `self = a.b`, so inner fields of `a.b` may change: **`a.b` stays valid, children are invalidated**
- In `a:method()`, `self = a`, so `a.b` may change: **`a.b` is invalidated**

### Pure-function whitelist

Some functions never mutate their arguments (e.g. `print`, `tostring`, `pairs`, `math.floor`). oLua ships a built-in whitelist; matching calls do not invalidate caches.

You can also add custom pure functions via `-opt_table_access_pure_funcs` (**regex**, comma-separated):

```bash
./oLua -input file.lua -output out.lua -opt_table_access \
       -opt_table_access_pure_funcs "log_.*,debug_.*,trace_.*"
```

The default `log_.*` matches log helpers such as `log_info`, `log_error`, and `log_debug`.

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-opt_table_access` | `false` | Enable table access optimization |
| `-opt_table_access_threshold` | `2` | Minimum read count to trigger optimization (at least 2) |
| `-opt_table_access_pure_funcs` | `log_.*` | Comma-separated pure-function regex list |
| `-opt_table_access_global` | `false` | Also optimize `_G.xxx` (off by default; hurts readability) |

### Safety guarantees

Table-access optimization is based on strict read/write analysis and is **safe** for:
- Mixed if/else/while/for/repeat/do-end compound statements
- Consecutive reads across statements
- Auto-refresh after a call invalidates a parent table
- `:method()` self invalidation semantics
- Repeated accesses inside multi-line table constructors
- Nested loops and nested conditions

And these guards avoid breaking code:
- Automatic rename on local name conflicts (e.g. `a_b_1`)
- Skip already-optimized lines (no double optimization)
- Parent/child path conflict detection (do not optimize both in the same round)
- Model “condition runs before body” for compound statements
- Conservatively break analysis on `goto`/`label`

## Optimize Lua table constructors

### Basics

Lua constructor literals write all keys at once, avoiding repeated hash rehashes and array part growth from scattered assignments. Folding `a.k = v` back into the constructor can cut runtime cost a lot.

For example:
```lua
local a = { a = 1, 2 }
a.b = 1
a["c"] = 2
a[3] = 3
a.d = { e = 4 }
a.d.f = 5
```
becomes:
```lua
local a = {['a']=1, 2, ['b']=1, ['c']=2, [3]=3, ['d']={['e']=4, ['f']=5}}
```

### Capabilities

Also driven by read/write analysis; no unsafe merges:

- **Allow unrelated statements in between**: skip `local x = 5`, `print(...)`, etc., and keep absorbing later `a.k = v`
- **Recursive nested constructor merge**: after `a.d = { e=1 }`, follow-ups like `a.d.f = 2` fold into the inner `a.d`
- **Last write wins for the same key**: `a.x=1; a.y=2; a.x=3` → `{['x']=3, ['y']=2}`
- **Leaf overwrite of a subtable**: `a.d = {e=1}` then `a.d = 5` merges to `d=5`
- **Non-local targets**: `s.o = {}` then `s.o.k = v` can merge (any static path root)
- **Array + named fields**: `{1, 2}` then `a.b = 3` → `{1, 2, ['b']=3}`
- **Multi-line RHS constructors**: absorbing `a.f = {\n 1,2,3 \n}` removes all related lines

### Stop rules (preserve semantics)

Stop absorbing as soon as any of these appear:

- **Read of the target table**: `y = a.c` or `if a.b then ...` — merging would turn a former `nil` field into a value
- **Write / mutating call on the target**: `a[b] = 5` (dynamic key), `f(a)` (may mutate `a`)
- **RHS reads/mutates the target**: `a.c = a.b + 1` or `a.c = f(a)` — merging would change evaluation timing
- **Dynamic key**: `a[var] = 5` is rejected when the key is not a compile-time constant
- **Loops / goto / label**: stop conservatively

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-opt_table_constructor` | `false` | Enable table constructor optimization |

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

Enable both and batch-process a directory (in-place):
```bash
./oLua -inputpath input_dir -opt_table_access -opt_table_constructor
```

In batch mode, a parse failure on one file does not abort the run; that file is skipped and others continue.

## Results

Benchmarks on PUC Lua 5.4.8 (best of 3 runs per scenario):

| Scenario | Description | Before | After | Speedup |
|----------|-------------|--------|-------|---------|
| Shallow multi-read | 5M game updates (`self.physics.velocity.x`, etc.) | 0.78 s | 0.40 s | **~1.95×** |
| Deep chain | 1M × 8-level `a.b.c.d.e.f.g.h`, 4 reads each | 0.088 s | 0.026 s | **~3.4×** |
| Business hotspot | 5000 rounds × 1000 rows, optimized only on hit branches | 0.32 s | 0.30 s | ~1.07× |

Typical business code (repeated `self.xxx.yyy.zzz`) sees about **1.5×~2×**; extreme deep access can reach **3×+**. Cold paths or rare accesses are left untouched.

Table constructor optimization:

|   | Before | After |
|---|--------|-------|
| table constructor | 3.3 s | 1.9 s |

## Testing

Unit tests:
```bash
go test ./...
```

Coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Benchmarks:
```bash
go test -bench=. -benchtime=1s -run="^$" ./...
```

## Also see
[lua family bucket](https://github.com/esrrhs/lua-family-bucket)
