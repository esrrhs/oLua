-- 覆盖率补充：深度嵌套场景，触发 stmtContainsWrite/stmtContainsRead 各分支

function test_nested_write_in_do()
    -- 外层读，do 块内嵌套 do 块有写
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    do
        do
            a.b = {}
        end
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_nested_write_in_repeat()
    -- repeat 块内有写
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    repeat
        a.b = {}
    until true
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_nested_write_in_for_generic()
    -- for-generic 内有写
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    for k, v in pairs(t) do
        a.b = {}
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_nested_write_in_for_numeric()
    -- for-numeric 内有写（从外层检测）
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    for i = 1, func1(a_b) do
        local z = 1
    end
    local w = a_b.e
    local v = a_b.f
end

function test_nested_read_in_all_blocks()
    -- 外层无足够读，但各种子块内有读（触发 stmtContainsRead 各分支）
    if cond then
        do
            local a_b = a.b -- opt by oLua
            local x = a_b.c
            local y = a_b.d
            local z = a_b.e
        end
    end
end

function test_nested_read_in_while()
    if cond then
        while a.b.active do
            local a_b = a.b -- opt by oLua
            local x = a_b.c
            local y = a_b.d
        end
    end
end

function test_nested_read_in_repeat()
    if cond then
        repeat
            local a_b = a.b -- opt by oLua
            local x = a_b.c
            local y = a_b.d
            local z = a_b.e
        until a.b.done
    end
end

function test_nested_read_in_for_numeric()
    if cond then
        for i = a.b.start, a.b.stop, a.b.step do
            local x = 1
        end
    end
end

function test_nested_read_in_for_generic()
    if cond then
        for k, v in pairs(a.b.data) do
            local a_b = a.b -- opt by oLua
            local x = a_b.c
            local y = a_b.d
        end
    end
end

function test_nested_call_in_parens()
    -- 嵌套调用在括号内
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = (func1(a_b))
    local w = a_b.e
    local v = a_b.f
end

function test_nested_call_in_table_constructor_key()
    -- 嵌套调用在 table constructor 的值中
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local t = {[func1(a_b)] = 1}
    local z = a_b.e
    local w = a_b.f
end

function test_dynamic_key_fallback()
    -- 动态 key 场景，getExprPath 返回 false
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = a[i].b
    local w = a_b.e
end
