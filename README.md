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


## Localize globals / library functions

### Basics

Hot functions often call the same globals or library functions many times (`print`, `string.format`, `math.floor`, …). Each call looks up the name through `_ENV`. Hoisting them into locals at function entry removes those lookups.

For example:
```lua
function hot()
    print(1)
    print(2)
    print(string.format("%d", 3))
    print(string.format("%d", 4))
end
```
becomes:
```lua
function hot()
    local print, format = print, string.format -- opt by oLua
    print(1)
    print(2)
    print(format("%d", 3))
    print(format("%d", 4))
end
```

### Rules (v1)

- Only a fixed whitelist is localized (common bare globals, `math.*`, `string.*`, `table.concat`, `os.clock` / `os.time` / `os.date`)
- A name is localized only when it is read at least `-opt_localize_globals_threshold` times in that function
- Skip when the name (or library root) is already local, when the function assigns that global/parent, or when the function rebinds `_ENV` / `_G`
- Method calls such as `o:format()` / field reads `o.format` are not treated as `string.format`
- Nested function bodies are analyzed separately; existing `-- opt by oLua` lines are left untouched

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-opt_localize_globals` | `false` | Enable localize globals / library functions |
| `-opt_localize_globals_threshold` | `2` | Minimum read count to trigger localization |


## Loop-invariant code motion (LICM)

### Basics

When a static table path is read many times inside a loop and nothing in that loop can write or invalidate it, oLua hoists one local before the loop header and rewrites the body (and while/repeat condition) reads.

For example:
```lua
function hot(a)
    for i = 1, 100 do
        local x = a.b.c + i
        local y = a.b.d + i
        local z = a.b.e + i
    end
end
```
becomes:
```lua
function hot(a)
    local a_b = a.b -- opt by oLua
    for i = 1, 100 do
        local x = a_b.c + i
        local y = a_b.d + i
        local z = a_b.e + i
    end
end
```

### Rules (v1)

- Only static dotted paths from `getExprPath` (no dynamic keys)
- Body (plus while/repeat condition) reads must reach `-opt_licm_threshold` (clamped to ≥ 2)
- Skip the loop when it contains `goto` / `label` / `break`
- Skip paths that include a for-loop iterator binder
- Skip when the loop writes or invalidates the path, or contains any non-pure call
- Nested function bodies are analyzed separately; one hoist per optimization round

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-opt_licm` | `false` | Enable loop-invariant code motion |
| `-opt_licm_threshold` | `2` | Minimum body read count to hoist (at least 2) |



## Constant folding / dead code elimination

### Basics

Pure constant arithmetic, relational, and boolean expressions are folded to literals. `if true` / `if false` branches are eliminated (Lua-truthiness caveat: only boolean `true`/`false`, not `if 0`).

For example:
```lua
function f()
    local x = 2 + 3 * 4
    if false then
        print("dead")
    else
        print("alive")
    end
end
```
becomes:
```lua
function f()
    local x = 14 -- opt by oLua
        print("alive") -- opt by oLua
end
```

### Rules (v1)

- Fold pure const arith (`+ - * / // % ^` unary `-`), relational (`== ~= < > <= >=`), and bool (`and or not`)
- Skip division/modulo by zero; skip bitwise, concat, and length
- DCE only for `if true` / `if false` (not `if 0` / other truthy consts)
- Skip DCE when the if then/else contains `goto` / `break` / `label`
- One fold or one DCE per optimization round; nested functions analyzed separately

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-opt_const_fold` | `false` | Enable constant folding and if true/false DCE |


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
