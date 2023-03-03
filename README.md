# oLua
[<img src="https://img.shields.io/github/license/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/languages/top/esrrhs/oLua">](https://github.com/esrrhs/oLua)
[<img src="https://img.shields.io/github/actions/workflow/status/esrrhs/oLua/go.yml?branch=master">](https://github.com/esrrhs/oLua/actions)

一个聊胜于无的Lua优化工具。

## 优化点
- [x] 优化Lua的table访问
- [x] 优化Lua的table构造
- [ ] 优化Lua的字符串拼接

## 优化Lua的table访问
例如如下代码：
```lua
a.b = {}
a.b.data1 = "1"
a.b.data2 = "2"
a.b.data3 = "3"
```
a.b是一个table，每次访问a.b都会触发一次table的访问，这样会影响性能，所以可以优化为：
```lua
a.b = {}
local a_b = a.b
a_b.data1 = "1"
a_b.data2 = "2"
a_b.data3 = "3"
```

## 优化Lua的table构造
例如如下代码：
```lua
local a = {}
a.b = 1
a["c"] = 2
a[3] = 3
```
每次往a中添加元素可能会触发table的扩容，所以可以优化为：
```lua
local a = {["b"] = 1, ["c"] = 2, [3] = 3}
```

## 使用
编译：
```bash
go mod tidy
go build
```
运行，优化单个文件的table访问：
```bash
./oLua -input input/table_access.lua -output output/table_access.lua -opt_table_access
```
运行，优化单个文件的table构造：
```bash
./oLua -input input/table_construct.lua -output output/table_construct.lua -opt_table_construct
```
也可以优化目录下的所有文件，原地替换：
```bash
./oLua -inputpath input_dir -opt_table_access -opt_table_construct
```

## 其他
[lua全家桶](https://github.com/esrrhs/lua-family-bucket)
