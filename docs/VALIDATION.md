# 语义验证与 Lua 实际性能

## 本轮发现及修复

不是以 golden 文件通过作为正确性标准。先运行差分复现，再收紧优化器。
修复前捕获的最小反例（每个均已加入 `testdata/semantic`）：

- **零次循环**：`for i=1,0 do x=a.b.c; y=a.b.c end`，`a=nil`；原程序
  返回 0，旧 LICM 提前索引而报错。现在只外提可证明存在的私有不可变路径。
- **元表读副作用**：`a.__index` 每次增加计数并返回不同的 `b`，循环内两读；
  旧 LICM 将结果/lookup 次数从 `21/6` 变为 `6/1`。未知/元表对象现在跳过。
- **按函数名判断纯度错误**：局部 `print` 闭包将 `a.b.c` 改为 42；旧 access
  缓存返回 `1,1` 而非 `1,42`。不再把名字白名单当证明；调用/闭包场景跳过。
- **constructor 删除调用**：`local a={x=hit()}; a.x=hit()`（独立行），旧合并
  将两次调用变成一次，结果及调用计数从 `2,2` 变为 `1,1`。
- **DCE 泄漏作用域**：`local x=7; if true then local x=9 end; return x`，旧版
  返回 9；现在保留 `do/end`，仍返回 7。另覆盖闭包捕获、elseif、嵌套 do/return。
- **整数精度**：旧版把 `9007199254740993 == 9007199254740992` 折成 true；
  `9223372036854775807 // 3` 折成 `3074457345618258432` 而非
  `3074457345618258602`，其 `% 3` 从 1 变为 0。现在整数路径全程精确。
- **字符串误替换**：同一行 `local text="1+2"; local value=1+2`，旧版把 text
  改成 "3"。现在按代码 token 边界编辑，排除字符串/注释，歧义匹配跳过。
- **局部化与组合挂死**：间接重绑 `math.abs` 不能缓存；折叠已标记的调用行
  又被 localize 反复计数，曾导致重复插入。默认不缓存未经证明的全局环境；
  显式静态环境契约模式忽略已标记行，迭代上限 1,000。
- 补充防护：构造器不跨语句/生成的 do 块，不合并变量 RHS、数值/序列 key
  （`1`/`01`/`0x1` 等价但旧 tree 按文本区分），Lua 字符串转义与花括号 token，
  bracket-only 路径无实际改写时不继续插入，别名超过 100 个冲突仍继续寻找空名。

**重要兼容性变化**：`-opt_localize_globals` 单独开启现在保守 no-op。只有额外显式
传 `-unsafe_assume_static_globals`，并由调用者保证 `_ENV`、库表和函数是普通、
无 lookup 副作用且不会间接重绑时，才恢复旧白名单功能；这不属于默认语义保证。
access/LICM 也不再处理任意参数/全局表。旧 golden 对应安全策略更新，而非保留
旧的错误输出。私有表、零次循环、各类循环、constructor、DCE 均有正向改写断言。

## 可重现验证

已有本地运行时：Lua 5.4.7；没有为本轮安装额外运行时。Go 1.27.3 linux/amd64。

```bash
# 至少需要一个 64 位整数 Lua 5.3/5.4；缺失会失败，不会 skip。
go test -count=1 ./...
go test -run 'Test(Runtime|Safe|Static|BenchmarkWorkload)' -count=1 ./...
go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go vet ./...
go tool cover -func=coverage.out

# 显式指定多个版本时，每一个都必须存在且版本受支持。
OLUA_LUA=lua5.3:lua5.4 go test -count=1 ./...
```

- **37 个固定回归 × 全部 32 种开关组合 = 1,184 组/运行时**；固定 mask 顺序。
  每组执行优化前/后 Lua，比较返回个数、值、integer/float 类型、nil、布尔、
  stdout 和错误（只去掉受源码插行影响的报错行位置）。用返回值/输出显式
  暴露查表次数、闭包写入、调用计数等副作用，不尝试通用序列化 table/function。
- 每组再次完整优化，要求源码字节完全一致；优化/测试 helper 都有 1,000 轮上限。
- 803 个固定数值表达式网格，按 24 个一组比较 Lua 原/后值，单开 fold 和全开
  两种配置。覆盖十进制前导零、hex、大整数、溢出、混合浮点、负整除/取模和 -0。
- 额外在稳定环境契约下跑全 32 组合，强制断言全开时确实进行了 math.abs
  局部化；与私有表 LICM/access、constructor、fold/DCE 一起验证并检查幂等。
- 4 个 benchmark 工作负载也做差分 checksum 和幂等验证；契约模式另覆盖别名
  遮蔽、构造器命名字段和长字符串，不能把 unsafe 理解成允许错误的源码编辑。
- CI 强制安装 Lua 5.3/5.4，`OLUA_LUA` 显式要求两者，执行 race/coverage/vet
  和 benchmark smoke，保留覆盖率与 JSON。**CI 配置已改，但本轮未推送运行**；
  本地只验证了 5.4.7，不能宣称两个版本均已实际通过。
- 本地 `go test -race` 通过，Go 语句覆盖率 **61.8%**。覆盖率不是 Lua 行覆盖率；旧分析器保留的低层工具路径
  并非当前生产安全入口，并不是所有路径都已覆盖。

## 实际 Lua 性能与大小

```bash
python3 scripts/benchmark_lua.py --lua lua --output lua-benchmark.json
# CI/smoke（校验 checksum，不设不稳定的性能阈值）
python3 scripts/benchmark_lua.py --lua lua --warmups 1 --repeats 3 --batch 1
```

脚本在临时目录构建优化器，五个优化开关全开、保持默认安全策略，不覆盖已有
`./oLua`。每个 fixture 返回可重复执行的 workload 函数。Lua 内部 `os.clock`
测 CPU 时间，不含编译/启动；先核对 checksum，双方各预热 2 次，每轮各执行
3 次，交替先后顺序，共 7 轮，每批前完整 GC。JSON 保存全部样本、标准差、
环境、优化器源码/fixture SHA256、字节数。两边都包含同样的 checksum assert 开销。
源码大小包含注释及 `-- opt by oLua` 标记，不是 minify 大小。

实测：2026-11-10T20:19:25.293232+08:00（Asia/Shanghai，UTC+8）；
Linux 6.12.113+ x86_64，Intel Xeon，共享主机，Lua 5.4.7，Go 1.27.3。
原始记录见 [benchmark-results.json](benchmark-results.json)。

| workload | 原版中位 ms [min,max] | 优化后中位 ms [min,max] | 源码 bytes | 原/后中位比 |
|---|---:|---:|---:|---:|
| arithmetic_filter | 1.539 [1.445, 1.592] | 1.172 [1.130, 1.234] | 259 → 239 | 1.314× |
| config_reduce | 7.515 [7.334, 8.305] | 5.013 [4.754, 5.097] | 298 → 396 | 1.499× |
| constructor_records | 29.596 [29.110, 30.889] | 16.837 [16.647, 20.523] | 302 → 291 | 1.758× |
| invoice_batch | 2.432 [2.341, 2.495] | 2.402 [2.356, 2.502] | 670 → 670 | 1.012× |

`config_reduce` 是配置加权归约，`constructor_records` 是固定记录分配/消费，
`arithmetic_filter` 是配置分支数值内核；它们是代表性内核，不是生产应用端到端
数据。`invoice_batch` 用确定性生成账单数据，包含常见表/调用/分支/舍入/归约，
在当前证明能力之外，源码保持不变；其时差应视为噪声而非提速或退化结论。
共享机频率/调度/GC 会影响结果，有可见离群值；不能泛化成任意 Lua 的提速保证。
缓存还可能增大源码。本轮没有 LuaJIT、生产数据集或多个硬件的性能结论。

## 已知边界

- 证明只覆盖封闭、私有、字面量表子集；不做跨函数类型/别名/元表效果分析。
  局部作用域环境在证明时恢复，不把退出作用域的 local 当作仍然存在。
- 为保守保证整行插入安全，重名遮蔽、同一行多个语句、长字符串及不能精确编辑
  的路径常跳过；为 Lua 的 200-local 限制保留 10 个名字余量。
- 目标是 Lua 5.3/5.4 的 64 位整数/IEEE double，解析器不是完整 Lua 5.4 前端。
  不保证 debug 钩子/局部变量反射、错误行号/回溯文本、GC/内存分配时刻或资源耗尽
  在源码变换后不变。普通返回值、输出、错误内容和可观察业务副作用是回归目标。
- 浮点除法、整除/取模、幂及字符串比较不做跨运行时折叠。不存在整数精度到
  float64 的隐式降级；无法安全解析的大数字保持原样。
