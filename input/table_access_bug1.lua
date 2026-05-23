-- 测试：复合语句内同时有读和写时不应被错误替换

function test_if_contains_write_and_read()
    -- if 块内有对 a.b 的写，不应在外层替换 if 内部的 a.b
    local x = a.b.c
    local y = a.b.d
    if cond then
        a.b = {}        -- 写 a.b
        local z = a.b.e -- 写之后的读
    end
    local w = a.b.f
end

function test_while_contains_invalidating_call()
    -- while 条件中有函数调用使 target 失效
    local x = a.b.c
    local y = a.b.d
    while func1(a.b) do
        local z = a.b.e
    end
    local w = a.b.f
end
