-- 覆盖率100%测试：触发所有剩余未覆盖分支

-- getExprPath default 分支：a[1] 数字索引作为 key
function test_numeric_key()
    local a_b = a.b -- opt by oLua
    local x = a_b[1]
    local y = a_b[2]
    local z = a_b[3]
end

-- exprContainsPathRead via FuncCall.Function：target 作为被调用的函数
function test_target_as_function()
    -- a.b 本身被当作函数调用
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b(1, 2)
    local z = a_b.d
end

-- nestedCallInvalidates 全路径：通过 Operator 内嵌套、Parens 内嵌套、TableConstructor key 内嵌套
function test_nested_invalidate_all_paths()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    -- Operator 左右内嵌套调用
    local z = 1 + foo(a_b) - bar(a_b)
    local w = a_b.e
    local v = a_b.f
end

-- exprContainsFuncCallInvalidating: Receiver 内含使 target 失效的嵌套 FuncCall
function test_invalidating_in_receiver()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    -- receiver 是一个返回值，其内部 FuncCall 传了 a.b
    local z = foo(a_b):method()
    local w = a_b.e
    local v = a_b.f
end

-- analyzeBlockAccess: LHS 表达式中有函数调用使 target 失效 (line 445)
-- 虽然罕见但合法: a[foo(a.b)] = 1
function test_lhs_func_call_invalidates()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a[foo(a.b)].x = 1
    local z = a_b.e
    local w = a_b.f
end

-- analyzeBlockAccess: ForLoopNumeric 的 Cond 中有使 target 失效的函数调用
function test_for_cond_func_invalidates()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    for i = 1, foo(a_b), 1 do
        break
    end
    local z = a_b.e
    local w = a_b.f
end

-- analyzeBlockAccess: Return 中有函数调用使 target 失效 (line 549)
function test_return_func_invalidates()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    return foo(a_b)
end

-- stmtContainsRead: FuncCall 语句包含对 Receiver 和 Function 的读
function test_stmt_read_func_receiver()
    -- 嵌套场景：外层 if 块检测 stmtContainsRead 对内部 FuncCall
    if cond then
        local a_b = a.b -- opt by oLua
        a_b:method()
        a_b:other()
        a_b:third()
    end
end

-- stmtContainsRead: Return 语句含读
function test_stmt_read_return()
    if cond then
        local a_b = a.b -- opt by oLua
        return a_b.c, a_b.d, a_b.e
    end
end

-- stmtContainsRead: ForLoopGeneric 内 init 含读
function test_stmt_read_for_generic_init()
    if cond then
        for k, v in a.b.iter() do
            local a_b = a.b -- opt by oLua
            local x = a_b.c
            local y = a_b.d
        end
    end
end

-- optimizeBlock: If.Else 递归分支 (line 938)
function test_optimize_if_else()
    if cond then
        -- 空
    else
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        local z = a_b.e
    end
end
