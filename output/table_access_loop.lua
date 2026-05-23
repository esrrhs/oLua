-- Test loop-specific patterns

function test_while_loop()
    -- a.b is read in condition AND body. Should optimize.
    while a.b.active do
        local a_b = a.b -- opt by oLua
        local x = a_b.value
        local y = a_b.name
        -- process x and y
    end
end

function test_for_with_method_in_body()
    -- Loop body has a method call that invalidates
    for i = 1, 10 do
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        a_b:update()       -- invalidates a_b
        local z = a_b.e
        local w = a_b.f
    end
end

function test_loop_safe_reads()
    -- No writes inside loop, safe to cache before the loop
    for i = 1, 100 do
        local a_b = a.b -- opt by oLua
        local x = a_b.c + i
        local y = a_b.d + i
        local z = a_b.e + i
    end
end

function test_nested_loop()
    for i = 1, 10 do
        for j = 1, 10 do
            local a_b_c = a.b.c -- opt by oLua
            local x = a_b_c.d
            local y = a_b_c.e
            local z = a_b_c.f
        end
    end
end
