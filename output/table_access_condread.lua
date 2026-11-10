-- 测试：if 条件中的读不应被 body 中的写操作失效
-- 因为条件先于 body 执行

function test_if_cond_read_body_write()
    -- v.TLogEventID 被缓存后，if 条件中的读应该继续使用缓存
    -- 即使 if body 内的函数调用使 target 失效
    local x = v.TLogEventID
    local y = v.TLogEventID
    if v.TLogEventID == 301125 and ss.data then
        ss.data.gyroscope_status = v.reason or 0
    end
    if v.TLogEventID == 10902 or v.TLogEventID == 300262 then
        report_virtual_voucher_player_behavior(ss, v)
    end
    if v.TLogEventID == 10049 then
        report_user_appeal(ss, v.TLogEventID)
    end
end

function test_while_cond_read_body_write()
    -- while 条件中的读也是先执行的
    local x = a.b.c
    local y = a.b.d
    while a.b.active do
        func1(a)  -- 使 a.b 失效
        break
    end
    local z = a.b.e
    local w = a.b.f
end
