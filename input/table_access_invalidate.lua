-- 测试：func1(a.b) 不应使 a.b 缓存失效
-- 因为函数拿到的是 a.b 的值，不能修改 a 表上 b 字段的绑定

function test_pass_target_self_no_invalidate()
    -- func1(v.TLogEventID) 传入的是值，不能反向修改 v.TLogEventID
    local x = v.TLogEventID
    report(ss, v.TLogEventID, v.reason)
    if v.TLogEventID == 10049 then  -- 应该被替换为 v_TLogEventID
        print("yes")
    end
    local y = v.TLogEventID
end

function test_pass_parent_invalidates()
    -- func1(v) 传入 v 的引用，可以做 v.TLogEventID = xxx → 失效
    local x = v.TLogEventID
    local y = v.TLogEventID
    func1(v)
    local z = v.TLogEventID  -- 这里应该刷新
    local w = v.TLogEventID
end

function test_pass_target_table_no_invalidate()
    -- func1(a.b) 传入 a.b 的引用，不能修改 a 上的 b 字段
    -- a.b 缓存仍有效
    local x = a.b.c
    local y = a.b.d
    func1(a.b)
    local z = a.b.e  -- a.b 缓存应仍有效
    local w = a.b.f
end

function test_pass_parent_table_invalidates()
    -- func1(a) 传入 a 的引用，可以做 a.b = xxx → a.b 缓存失效
    local x = a.b.c
    local y = a.b.d
    func1(a)
    local z = a.b.e  -- 这里应该刷新
    local w = a.b.f
end

function test_method_on_target_no_invalidate()
    -- a.b:method() → self=a.b，不能修改 a 上 b 字段的绑定
    -- a.b 缓存仍有效
    local x = a.b.c
    local y = a.b.d
    a.b:method()
    local z = a.b.e  -- a.b 缓存应仍有效
    local w = a.b.f
end

function test_method_on_parent_invalidates_child()
    -- a:method() → self=a，可以做 self.b = xxx → a.b 缓存失效
    local x = a.b.c
    local y = a.b.d
    a:method()
    local z = a.b.e  -- 这里应该刷新
    local w = a.b.f
end
