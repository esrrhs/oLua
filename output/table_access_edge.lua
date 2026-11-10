-- Edge case tests

function test_self_assignment()
    -- a.b assigned to itself - should still work
    a.b = a.b
    local x = a.b.c
    local y = a.b.d
end

function test_multiple_targets()
    -- Multiple assignment targets
    a.b.c, a.b.d = 1, 2
    a.b.e = 3
    a.b.f = 4
end

function test_loop_with_reads()
    -- Reads inside a loop body
    for i = 1, 10 do
        local x = a.b.c
        local y = a.b.d
        local z = a.b.e
    end
end

function test_for_generic()
    -- Generic for loop
    for k, v in pairs(a.b) do
        local x = a.b.c
        local y = a.b.d
    end
end

function test_return_value()
    -- Return uses table access
    local x = a.b.c
    local y = a.b.d
    return a.b.e
end

function test_condition_reads()
    -- Table access in condition
    if a.b.c then
        local x = a.b.d
        local y = a.b.e
    end
end

function test_deep_chain()
    -- Deep chain optimization
    local x = a.b.c.d.e
    local y = a.b.c.d.f
    local z = a.b.c.d.g
end
