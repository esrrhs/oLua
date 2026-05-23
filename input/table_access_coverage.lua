-- 覆盖率补充测试：各种复合语句中的读写

function test_do_block_write()
    -- do...end 块内有写操作
    local x = a.b.c
    local y = a.b.d
    do
        a.b = {}
    end
    local z = a.b.e
    local w = a.b.f
end

function test_repeat_until()
    -- repeat...until 块内读写
    repeat
        local x = a.b.c
        local y = a.b.d
        local z = a.b.e
    until a.b.done
end

function test_for_generic_read()
    -- for...in 循环内读
    for k, v in pairs(t) do
        local x = a.b.c
        local y = a.b.d
        local z = a.b.e
    end
end

function test_nested_func_call_in_constructor()
    -- table constructor 中的嵌套函数调用使 target 失效
    local x = a.b.c
    local y = a.b.d
    local t = {value = func1(a.b)}
    local z = a.b.e
    local w = a.b.f
end

function test_nested_call_in_operator()
    -- 表达式运算中的嵌套函数调用
    local x = a.b.c
    local y = a.b.d
    local z = 1 + func1(a.b)
    local w = a.b.e
    local v = a.b.f
end

function test_while_write_in_body()
    -- while 循环体内有写操作
    local x = a.b.c
    local y = a.b.d
    while cond do
        a.b = {}
    end
    local z = a.b.e
    local w = a.b.f
end

function test_for_numeric_write()
    -- for 循环体内有写操作
    local x = a.b.c
    local y = a.b.d
    for i = 1, 10 do
        a.b = {}
    end
    local z = a.b.e
    local w = a.b.f
end

function test_if_cond_func_call()
    -- if 条件中有函数调用使 target 失效
    local x = a.b.c
    local y = a.b.d
    if func1(a.b) then
        local z = 1
    end
    local w = a.b.e
    local v = a.b.f
end

function test_return_with_reads()
    -- return 语句中多次读取
    local x = a.b.c
    return a.b.d, a.b.e, a.b.f
end

function test_do_block_read_only()
    -- do 块内只有读
    local x = a.b.c
    do
        local y = a.b.d
        local z = a.b.e
    end
    local w = a.b.f
end

function test_nested_func_in_func_expr()
    -- getHandler(a.b)() 形式：Function 表达式内嵌套调用
    local x = a.b.c
    local y = a.b.d
    getHandler(a.b)()
    local z = a.b.e
    local w = a.b.f
end

function test_parens_expr()
    -- 括号表达式中的读
    local x = (a.b.c)
    local y = (a.b.d)
    local z = (a.b.e)
end
