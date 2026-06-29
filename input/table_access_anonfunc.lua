-- 匿名函数表达式测试

function test_anon_func_value()
    -- 定义不立即执行 a.b = {}，缓存仍有效
    local x = a.b.c
    local y = a.b.d
    local f = function() a.b = {} end
    local z = a.b.e
    local w = a.b.f
end

function test_iife_invalidates()
    -- IIFE 立即执行，应使 a.b 失效，需刷新
    local x = a.b.c
    local y = a.b.d;
    (function() a.b = {} end)();
    local z = a.b.e
    local w = a.b.f
end

function test_iife_no_write()
    -- IIFE 但 body 内不写 target，缓存仍有效
    local x = a.b.c
    local y = a.b.d;
    (function() return 1 end)();
    local z = a.b.e
    local w = a.b.f
end

function test_anon_passed_to_callback()
    -- 匿名函数传入 callback，body 不被立即执行
    -- 当前保守处理：函数定义本身不会使 target 失效（取决于 callback 的实现）
    local x = a.b.c
    local y = a.b.d
    callback(function() a.b = {} end)
    local z = a.b.e
    local w = a.b.f
end

function test_nested_function_decl()
    -- local function 形式（语法糖）
    local x = a.b.c
    local y = a.b.d
    local function inner()
        a.b = {}
    end
    local z = a.b.e
    local w = a.b.f
end
