-- Edge case tests

function test_self_assignment()
    -- a.b assigned to itself - should still work
    a.b = a.b
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
end

function test_multiple_targets()
    -- Multiple assignment targets
    local a_b = a.b -- opt by oLua
    a_b.c, a_b.d = 1, 2
    a_b.e = 3
    a_b.f = 4
end

function test_loop_with_reads()
    -- Reads inside a loop body
    for i = 1, 10 do
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        local z = a_b.e
    end
end

function test_for_generic()
    -- Generic for loop
    for k, v in pairs(a.b) do
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
    end
end

function test_return_value()
    -- Return uses table access
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    return a_b.e
end

function test_condition_reads()
    -- Table access in condition
    if a.b.c then
        local a_b = a.b -- opt by oLua
        local x = a_b.d
        local y = a_b.e
    end
end

function test_deep_chain()
    -- Deep chain optimization
    local a_b_c_d = a.b.c.d -- opt by oLua
    local x = a_b_c_d.e
    local y = a_b_c_d.f
    local z = a_b_c_d.g
end
