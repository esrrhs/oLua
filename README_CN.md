# oLua
[<img src="https://img.shields.io/github/license/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/languages/top/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/actions/workflow/status/esrrhs/oLua/go.yml?branch=master">](https://github.com/esrrhs/oLua/actions)

[English](./README.md) | [中文](./README_CN.md)

一个聊胜于无的 Lua 优化工具：基于读写分析，安全地加速 table 访问、table 构造、热路径上的全局/库函数查找、循环不变量表路径外提，以及常量折叠 / 死代码消除。

## 优化点
- [x] 优化 Lua 的 table 访问（基于读写分析，安全可靠）
- [x] 优化 Lua 的 table 构造
- [x] 局部化全局 / 库函数（`-opt_localize_globals`）
- [x] 循环不变量外提 / LICM（`-opt_licm`）
- [x] 常量折叠 / 死代码消除（`-opt_const_fold`）


## 优化 Lua 的 table 访问

### 基本原理

Lua 中每次 `a.b` 都会触发一次 table 查找。如果在一个作用域内多次访问同一个表路径，可以缓存到 local 变量减少查找次数。

例如：
```lua
local x = a.b.c
local y = a.b.d
local z = a.b.e
```
优化为：
```lua
local a_b = a.b -- opt by oLua
local x = a_b.c
local y = a_b.d
local z = a_b.e
```

### 读写分析

不同于简单的模式匹配，oLua 使用**读写分析**（Read/Write Analysis）来判断什么时候缓存仍然有效：

- **写操作会使缓存失效**：`a.b = {}`、`a = ...`（父级写）、`func1(a)`（函数可能修改 a 的字段）等
- **写操作之后的读，需要刷新缓存**

例如：
```lua
local x = a.b.c
local y = a.b.d
func1(a)         -- 函数可能修改 a，使 a.b 缓存失效
local z = a.b.e
local w = a.b.f
```
优化为：
```lua
local a_b = a.b -- opt by oLua
local x = a_b.c
local y = a_b.d
func1(a)
a_b = a.b -- opt by oLua（自动刷新）
local z = a_b.e
local w = a_b.f
```

### 失效规则细节

- `func1(a.b)` 传入的是 `a.b` 的**值引用**，不能修改 `a` 表上 `b` 字段的绑定，所以 **a.b 缓存仍有效**
- `func1(a.b)` 中函数可以修改 `a.b` 内部的字段（如 `arg.x = 1`），所以 **a.b.x 等子级缓存会失效**
- `func1(a)` 传入 `a` 整个表的引用，可以做 `a.b = xxx`，所以 **a.b 缓存失效**
- `a.b:method()` 中 `self = a.b`，可以修改 `a.b` 内部字段，所以 **a.b 仍有效，子级失效**
- `a:method()` 中 `self = a`，可以修改 `a.b`，所以 **a.b 失效**

### 纯函数白名单

某些函数确实不修改参数（如 `print`、`tostring`、`pairs`、`math.floor`），oLua 内置了常用纯函数白名单。匹配的调用不会使缓存失效。

用户也可以通过 `-opt_table_access_pure_funcs` 添加自定义纯函数（**正则表达式**，逗号分隔）：

```bash
./oLua -input file.lua -output out.lua -opt_table_access \
       -opt_table_access_pure_funcs "log_.*,debug_.*,trace_.*"
```

默认值 `log_.*` 匹配 `log_info`, `log_error`, `log_debug` 等日志函数。

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-opt_table_access` | `false` | 启用 table 访问优化 |
| `-opt_table_access_threshold` | `2` | 触发优化的最少读取次数（最小为 2） |
| `-opt_table_access_pure_funcs` | `log_.*` | 纯函数正则表达式列表（逗号分隔） |
| `-opt_table_access_global` | `false` | 是否优化 `_G.xxx` 形式（默认关闭，开启后影响可读性） |

### 安全保证

oLua 的 table 访问优化基于严格的读写分析，对以下场景**安全**：
- 各种 if/else/while/for/repeat/do-end 复合语句的混合
- 跨语句的连续读取
- 函数调用使父级表失效后的自动刷新
- 方法调用 `:method()` 的 self 失效语义
- 多行 table 构造器中的多次访问
- 嵌套循环和嵌套条件

并通过以下机制避免破坏代码：
- 变量名冲突自动检测（生成 `a_b_1` 等替代名）
- 跳过已优化的代码行（避免重复优化）
- 父子路径冲突检测（同一轮不同时优化父和子）
- 复合语句中"条件先于 body 执行"的语义建模
- 保守处理 `goto`/`label`（控制流跳转打断分析）

## 优化 Lua 的 table 构造

### 基本原理

Lua 的构造器字面量一次性把所有键值写入，避免多次赋值触发的 hash 表 rehash 与 array 部分扩容。所以把散落的 `a.k = v` 尽量回收进构造器可以显著减少运行时开销。

例如如下代码：
```lua
local a = { a = 1, 2 }
a.b = 1
a["c"] = 2
a[3] = 3
a.d = { e = 4 }
a.d.f = 5
```
优化为：
```lua
local a = {['a']=1, 2, ['b']=1, ['c']=2, [3]=3, ['d']={['e']=4, ['f']=5}}
```

### 算法能力

同样基于读写分析，不做危险合并：

- **允许无关语句穿插**：中间夹了 `local x = 5`、`print(...)` 等不涉及目标表的语句，会跳过继续吸收后续 `a.k = v`
- **嵌套构造器递归合并**：`a.d = { e=1 }` 之后的 `a.d.f = 2`、`a.d.g = 3` 会被递归折进 `a.d` 内层
- **同键最后写入胜出**：`a.x=1; a.y=2; a.x=3` 合并为 `{['x']=3, ['y']=2}`，前面的被丢弃
- **子表被叶子覆盖**：`a.d = {e=1}` 后跟 `a.d = 5` 正确合并为 `d=5`
- **非 local 目标**：`s.o = {}` 后跟 `s.o.k = v` 也能合并（任意静态路径根）
- **数组位 + 命名混合**：`{1, 2}` 后跟 `a.b = 3` 合并为 `{1, 2, ['b']=3}`
- **多行 RHS 构造器**：`a.f = {\n 1,2,3 \n}` 被吸收时自动删除所有相关行

### 停止规则（保证语义）

一旦扫描到以下情况就停止吸收，避免语义变化：

- **读目标表**：`y = a.c` 或 `if a.b then ...` —— 合并会让原本 `nil` 的字段变成有值
- **写目标或修改目标的调用**：`a[b] = 5`（动态 key）、`f(a)`（可能改 a）—— 直接触发失效
- **RHS 读/改目标**：`a.c = a.b + 1` 或 `a.c = f(a)` —— 合并会改变求值时刻
- **动态 key**：`a[var] = 5` 不接受编译期非常量的 key
- **循环 / goto / label**：保守停止

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-opt_table_constructor` | `false` | 启用 table 构造优化 |


## 局部化全局 / 库函数

### 基本原理

热函数里反复调用同一全局或库函数（`print`、`string.format`、`math.floor` 等）时，每次都会经 `_ENV` 查表。在函数入口提升为 local 可去掉这些查找。

例如：
```lua
function hot()
    print(1)
    print(2)
    print(string.format("%d", 3))
    print(string.format("%d", 4))
end
```
优化为：
```lua
function hot()
    local print, format = print, string.format -- opt by oLua
    print(1)
    print(2)
    print(format("%d", 3))
    print(format("%d", 4))
end
```

### 规则（v1）

- 仅处理固定白名单（常用裸全局、`math.*`、`string.*`、`table.concat`、`os.clock` / `os.time` / `os.date`）
- 同一函数内读取次数达到 `-opt_localize_globals_threshold` 才提升
- 若该名（或库根）已是 local、函数内对全局/父路径有赋值、或重绑定 `_ENV` / `_G`，则跳过
- 方法调用 `o:format()` / 字段 `o.format` 不视为 `string.format`
- 嵌套函数体单独分析；已有 `-- opt by oLua` 行不改写

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-opt_localize_globals` | `false` | 启用全局 / 库函数局部化 |
| `-opt_localize_globals_threshold` | `2` | 触发局部化的最少读取次数 |


## 循环不变量外提（LICM）

### 基本原理

当循环内多次读取同一个静态表路径，且循环内没有任何写/失效时，oLua 在循环头之前插入 local，并改写循环体（以及 while/repeat 条件）中的读取。

例如：
```lua
function hot(a)
    for i = 1, 100 do
        local x = a.b.c + i
        local y = a.b.d + i
        local z = a.b.e + i
    end
end
```
优化为：
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

### 规则（v1）

- 仅处理 `getExprPath` 得到的静态点分路径（动态 key 不支持）
- 循环体（含 while/repeat 条件）读取次数达到 `-opt_licm_threshold`（下限 2）
- 循环内出现 `goto` / `label` / `break` 则整段跳过
- 路径含 for 迭代变量则跳过
- 循环内对该路径有写/失效，或存在非纯函数调用，则跳过
- 嵌套函数体单独分析；每轮只外提一次

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-opt_licm` | `false` | 启用循环不变量外提 |
| `-opt_licm_threshold` | `2` | 触发外提的最少读取次数（最小为 2） |



## 常量折叠 / 死代码消除

### 基本原理

对纯常量算术、关系、布尔表达式做折叠；对 `if true` / `if false` 做死代码消除（注意 Lua 真值：只处理布尔 `true`/`false`，不会对 `if 0` 做 DCE）。

例如：
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
优化为：
```lua
function f()
    local x = 14 -- opt by oLua
        print("alive") -- opt by oLua
end
```

### 规则（v1）

- 折叠纯常量算术（`+ - * / // % ^` 与一元 `-`）、关系（`== ~= < > <= >=`）、布尔（`and or not`）
- 跳过除零/取模零；不折叠位运算、连接、长度运算
- DCE 仅针对 `if true` / `if false`（不对 `if 0` 等真值常量）
- 若 if 的 then/else 含 `goto` / `break` / `label` 则跳过 DCE
- 每轮只做一次折叠或一次 DCE；嵌套函数单独分析

### 命令行参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-opt_const_fold` | `false` | 启用常量折叠与 if true/false DCE |


## 使用

编译：
```bash
go mod tidy
go build
```

优化单个文件的 table 访问：
```bash
./oLua -input input.lua -output output.lua -opt_table_access
```

优化单个文件的 table 构造：
```bash
./oLua -input input.lua -output output.lua -opt_table_constructor
```

局部化全局 / 库函数：
```bash
./oLua -input input.lua -output output.lua -opt_localize_globals
```

循环不变量外提：
```bash
./oLua -input input.lua -output output.lua -opt_licm
```

常量折叠 / DCE：
```bash
./oLua -input input.lua -output output.lua -opt_const_fold
```

也可以同时启用多种优化、批量处理目录（原地替换）：
```bash
./oLua -inputpath input_dir -opt_table_access -opt_table_constructor -opt_localize_globals -opt_licm -opt_const_fold
```

批量模式下，单个文件解析失败不会终止整个流程，会跳过该文件继续处理其他文件。

## 效果

实测对比（PUC Lua 5.4.8，每个场景跑 3 次取最快）：

| 场景 | 描述 | 优化前 | 优化后 | 提速 |
|------|------|--------|--------|------|
| 浅层多读 | 500 万次游戏 update（`self.physics.velocity.x` 等典型字段访问） | 0.78 s | 0.40 s | **~1.95×** |
| 深层链 | 100 万次 8 层链 `a.b.c.d.e.f.g.h`，每次 4 读 | 0.088 s | 0.026 s | **~3.4×** |
| 业务热点 | 5000 轮 × 1000 条数据，分支命中后才优化 | 0.32 s | 0.30 s | ~1.07× |

业务代码典型场景（重复访问 `self.xxx.yyy.zzz`）大约 **1.5×~2× 提速**；极端深层访问可达 **3×+**；冷路径或访问次数少的代码不会被改动也不会有副作用。

table 构造优化：

|   | 优化前   | 优化后  |
|---|-------|------|
| table 构造 | 3.3 s | 1.9 s |

## 测试

运行单元测试：
```bash
go test ./...
```

查看代码覆盖率：
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

运行性能基准测试：
```bash
go test -bench=. -benchtime=1s -run="^$" ./...
```

## 其他
[lua全家桶](https://github.com/esrrhs/lua-family-bucket)
