-- 覆盖率补充测试：各种复合语句中的读写

function test_do_block_write()
    -- do...end 块内有写操作
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    do
        a.b = {}
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_repeat_until()
    -- repeat...until 块内读写
    repeat
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        local z = a_b.e
    until a.b.done
end

function test_for_generic_read()
    -- for...in 循环内读
    for k, v in pairs(t) do
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        local z = a_b.e
    end
end

function test_nested_func_call_in_constructor()
    -- table constructor 中的嵌套函数调用使 target 失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local t = {value = func1(a_b)}
    local z = a_b.e
    local w = a_b.f
end

function test_nested_call_in_operator()
    -- 表达式运算中的嵌套函数调用
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = 1 + func1(a_b)
    local w = a_b.e
    local v = a_b.f
end

function test_while_write_in_body()
    -- while 循环体内有写操作
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    while cond do
        a.b = {}
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_for_numeric_write()
    -- for 循环体内有写操作
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    for i = 1, 10 do
        a.b = {}
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_if_cond_func_call()
    -- if 条件中有函数调用使 target 失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    if func1(a_b) then
        local z = 1
    end
    local w = a_b.e
    local v = a_b.f
end

function test_return_with_reads()
    -- return 语句中多次读取
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    return a_b.d, a_b.e, a_b.f
end

function test_do_block_read_only()
    -- do 块内只有读
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    do
        local y = a_b.d
        local z = a_b.e
    end
    local w = a_b.f
end

function test_nested_func_in_func_expr()
    -- getHandler(a.b)() 形式：Function 表达式内嵌套调用
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    getHandler(a_b)()
    local z = a_b.e
    local w = a_b.f
end

function test_parens_expr()
    -- 括号表达式中的读
    local a_b = a.b -- opt by oLua
    local x = (a_b.c)
    local y = (a_b.d)
    local z = (a_b.e)
end
