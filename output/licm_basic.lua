function test_for_hoist()
    local a_b = a.b -- opt by oLua
    for i = 1, 100 do
        local x = a_b.c + i
        local y = a_b.d + i
        local z = a_b.e + i
    end
end

function test_while_hoist()
    local a_b = a.b -- opt by oLua
    while a_b.active do
        local x = a_b.value
        local y = a_b.name
    end
end

function test_repeat_hoist()
    local cfg_opts = cfg.opts -- opt by oLua
    repeat
        local x = cfg_opts.n
        local y = cfg_opts.m
    until done
end
