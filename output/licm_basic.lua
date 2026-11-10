function test_for_hoist()
    for i = 1, 100 do
        local x = a.b.c + i
        local y = a.b.d + i
        local z = a.b.e + i
    end
end

function test_while_hoist()
    while a.b.active do
        local x = a.b.value
        local y = a.b.name
    end
end

function test_repeat_hoist()
    repeat
        local x = cfg.opts.n
        local y = cfg.opts.m
    until done
end
