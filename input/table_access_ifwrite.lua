-- Focused test: if block containing writes to children

function test_if_child_write()
    -- a.b.c = ... is NOT a write to a.b (only invalidates a.b.c)
    -- So a.b should be cacheable across the entire function
    local x = a.b.c
    local y = a.b.d
    if cond then
        a.b.e = 100  -- writes a.b.e, but NOT a.b itself
        a.b.f = 200  -- writes a.b.f, but NOT a.b itself
    end
    local z = a.b.g
    local w = a.b.h
end

function test_if_direct_write()
    -- a.b = ... IS a write to a.b (invalidates)
    local x = a.b.c
    local y = a.b.d
    if cond then
        a.b = {}  -- writes a.b directly
    end
    local z = a.b.e
    local w = a.b.f
end
