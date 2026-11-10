-- Test loop-specific patterns

function test_while_loop()
    -- a.b is read in condition AND body. Should optimize.
    while a.b.active do
        local x = a.b.value
        local y = a.b.name
        -- process x and y
    end
end

function test_for_with_method_in_body()
    -- Loop body has a method call that invalidates
    for i = 1, 10 do
        local x = a.b.c
        local y = a.b.d
        a.b:update()       -- invalidates a.b
        local z = a.b.e
        local w = a.b.f
    end
end

function test_loop_safe_reads()
    -- No writes inside loop, safe to cache before the loop
    for i = 1, 100 do
        local x = a.b.c + i
        local y = a.b.d + i
        local z = a.b.e + i
    end
end

function test_nested_loop()
    for i = 1, 10 do
        for j = 1, 10 do
            local x = a.b.c.d
            local y = a.b.c.e
            local z = a.b.c.f
        end
    end
end
