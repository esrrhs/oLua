-- Focused test: if block containing writes to children

function test_if_child_write()
    -- a.b.c = ... is NOT a write to a.b (only invalidates a.b.c)
    -- So a.b should be cacheable across the entire function
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    if cond then
        a_b.e = 100  -- writes a_b.e, but NOT a_b itself
        a_b.f = 200  -- writes a_b.f, but NOT a_b itself
    end
    local z = a_b.g
    local w = a_b.h
end

function test_if_direct_write()
    -- a.b = ... IS a write to a.b (invalidates)
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    if cond then
        a.b = {}  -- writes a.b directly
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end
