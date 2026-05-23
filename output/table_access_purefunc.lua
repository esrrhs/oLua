-- 测试纯函数白名单：这些函数不会使 target 失效

function test_print_no_invalidate()
    -- print 是纯函数，传入 a.b 不失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    print(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_type_no_invalidate()
    -- type 是纯函数
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local t = type(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_pairs_no_invalidate()
    -- pairs 是纯函数
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    for k, v in pairs(a_b) do end
    local z = a_b.e
    local w = a_b.f
end

function test_tostring_no_invalidate()
    -- tostring 是纯函数
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local s = tostring(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_math_floor_no_invalidate()
    -- math.floor 是纯函数
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local n = math.floor(a_b.value)
    local z = a_b.e
    local w = a_b.f
end

function test_log_pattern_no_invalidate()
    -- log_* 匹配用户白名单正则
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    log_info(a_b)
    log_error(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_unknown_func_invalidates()
    -- 非白名单函数仍然使 target 失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    modify_table(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_method_call_still_invalidates()
    -- 方法调用不受白名单影响（self 传入 = 写）
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a_b:doSomething()
    local z = a_b.e
    local w = a_b.f
end
