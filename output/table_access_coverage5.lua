-- 覆盖率100%：触发 nestedCallInvalidates.TableConstructor.Vals 和 analyzeBlockAccess WhileCond

function test_nested_call_in_tc_val()
    -- TableConstructor val 位置含使 target 失效的嵌套调用
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local t = {a = 1, b = foo(a_b)}
    local z = a_b.e
    local w = a_b.f
end

function test_while_cond_invalidates()
    -- while 条件中有函数调用使 target 失效（analyzeBlockAccess line 510）
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    while foo(a_b) do
        break
    end
    local z = a_b.e
    local w = a_b.f
end

function test_repeat_cond_invalidates()
    -- repeat 条件中有函数调用使 target 失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    repeat
        break
    until foo(a_b)
    local z = a_b.e
    local w = a_b.f
end

function test_expr_func_call_in_function_expr()
    -- exprContainsFuncCallInvalidating: FuncCall.Function 内含递归 funcCall
    -- obj.method()(a.b) 形式
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = obj.method()(a_b)
    local w = a_b.e
    local v = a_b.f
end

function test_expr_func_call_in_args_recursive()
    -- exprContainsFuncCallInvalidating: Args 内含递归 funcCall
    -- foo(bar(a.b)) 形式
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = foo(bar(a_b))
    local w = a_b.e
    local v = a_b.f
end
