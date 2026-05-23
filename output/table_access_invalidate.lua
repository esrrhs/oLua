-- 测试：func1(a.b) 不应使 a.b 缓存失效
-- 因为函数拿到的是 a.b 的值，不能修改 a 表上 b 字段的绑定

function test_pass_target_self_no_invalidate()
    -- func1(v.TLogEventID) 传入的是值，不能反向修改 v.TLogEventID
    local v_TLogEventID = v.TLogEventID -- opt by oLua
    local x = v_TLogEventID
    report(ss, v_TLogEventID, v.reason)
    if v_TLogEventID == 10049 then  -- 应该被替换为 v_TLogEventID
        print("yes")
    end
    local y = v_TLogEventID
end

function test_pass_parent_invalidates()
    -- func1(v) 传入 v 的引用，可以做 v.TLogEventID = xxx → 失效
    local v_TLogEventID = v.TLogEventID -- opt by oLua
    local x = v_TLogEventID
    local y = v_TLogEventID
    func1(v)
    v_TLogEventID = v.TLogEventID -- opt by oLua
    local z = v_TLogEventID  -- 这里应该刷新
    local w = v_TLogEventID
end

function test_pass_target_table_no_invalidate()
    -- func1(a.b) 传入 a.b 的引用，不能修改 a 上的 b 字段
    -- a.b 缓存仍有效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    func1(a_b)
    local z = a_b.e  -- a_b 缓存应仍有效
    local w = a_b.f
end

function test_pass_parent_table_invalidates()
    -- func1(a) 传入 a 的引用，可以做 a.b = xxx → a.b 缓存失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    func1(a)
    a_b = a.b -- opt by oLua
    local z = a_b.e  -- 这里应该刷新
    local w = a_b.f
end

function test_method_on_target_no_invalidate()
    -- a.b:method() → self=a.b，不能修改 a 上 b 字段的绑定
    -- a.b 缓存仍有效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a_b:method()
    local z = a_b.e  -- a_b 缓存应仍有效
    local w = a_b.f
end

function test_method_on_parent_invalidates_child()
    -- a:method() → self=a，可以做 self.b = xxx → a.b 缓存失效
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a:method()
    a_b = a.b -- opt by oLua
    local z = a_b.e  -- 这里应该刷新
    local w = a_b.f
end
