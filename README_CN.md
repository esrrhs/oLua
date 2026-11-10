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


## 先保证语义，再考虑提速

所有优化默认关闭，只改函数体，不改顶层 chunk。运行时差分回归优先于旧文本
快照；不能证明安全的场景跳过，不能凭路径未写或“纯函数名字”推断安全。

- **table access / LICM**：仅处理函数最外层新建、纯字面量、不可变的私有
  local 表，且所有路径及中间表都已知存在。含调用、外部输入/闭包、表写、
  遮蔽、泛型迭代器等场景不处理。缓存声明支配所有使用点；零次循环也不
  提前触发 nil 错误或 `__index`。阈值分别是
  `-opt_table_access_threshold` / `-opt_licm_threshold`，默认及最小值均为 2。
  旧 `pure_funcs` / `_G` 参数保留兼容，但不能绕过安全证明。
- **constructor**：只合并新建 local 表后的相邻、独立单行、纯字面量写入；
  不跨越语句，不合并 RHS 调用/变量读取、全局/字段目标或数值/序列 key。
- **global / library localize**：单独开 `-opt_localize_globals` 现在会保守跳过。
  因为 `_ENV`、库表及函数可以重绑或带元方法，目前没有静态稳定性证明。
  仅当调用者能保证环境/库无元方法、无间接重绑且始终稳定，才可额外显式开
  `-unsafe_assume_static_globals` 恢复白名单局部化（**不是语义保证**）。
  阈值 `-opt_localize_globals_threshold` 默认 2。
- **const fold / DCE**：`-opt_const_fold` 保留 Lua 64 位整数精度/回绕、精确
  整除/取模和比较；消除 `if true/false` 时保留 `do/end` 词法作用域和 return。
  替换基于代码 token，不误改字符串/注释，结果保留括号。混合超大整数浮点
  比较、字符串常量、浮点除法/整除/取模/幂、零除和非有限结果保留到运行时。
  同行多 if、跳转分支及不能精确编辑的场景跳过。

支持目标为 Lua 5.3/5.4（64 位整数、IEEE double）；解析器并不覆盖所有 Lua 5.4
语法（如 local attributes）。有意收紧了此前未经证明的优化；旧快照也对应更新，
不是把错误结果当标准。另有“安全场景确实发生改写”的正向测试，避免全部 no-op。

管线：const fold/DCE → constructor → localize → LICM → table access。
重解析最多 1,000 轮，未收敛会报错，避免多优化组合挂死。

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

## 测试、覆盖率与 Lua 实际性能

测试必须有 Lua 5.3 或 5.4，缺失会失败，不会静默跳过。`OLUA_LUA` 可设单个
可执行文件，或用冒号分隔显式要求多个版本。

```bash
go test -count=1 ./...
go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go vet ./...
go tool cover -func=coverage.out
python3 scripts/benchmark_lua.py --lua lua --output lua-benchmark.json
```

详见 [验证说明与实测数据](docs/VALIDATION.md)：返回值（含类型及 nil）、输出、
副作用和错误差分，五个开关的全部 32 种组合，重复优化幂等，以及数值边界网格。
Python 工具实际执行 Lua，检查 checksum，预热、交替配对多次测量 CPU 时间，
同时报告源码字节数和波动。工作负载内核的收益不代表生产应用整体提速。
CI 强制安装 Lua 5.3/5.4，执行 race/覆盖率和基准 smoke，并保留报告。

## 其他
[lua全家桶](https://github.com/esrrhs/lua-family-bucket)
