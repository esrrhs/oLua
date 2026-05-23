-- Test advanced table access optimization scenarios

function test_basic_reads()
    -- Basic consecutive reads (should optimize a.b)
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = a_b.e
end

function test_write_interruption()
    -- Reads interrupted by direct write
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a.b = {}           -- write: invalidates a.b cache
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_func_call_invalidation()
    -- Function call with ancestor argument invalidates
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    func1(a)           -- passes 'a' to function, invalidates a.b
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_method_call_invalidation()
    -- Method call on target invalidates it
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    a_b:doSomething()  -- a_b is self, written
    local y = a_b.d
    local z = a_b.e
end

function test_nested_scope()
    -- Reads inside if block
    if cond then
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        local z = a_b.e
    end
end

function test_write_in_branch()
    -- Write inside a branch invalidates outer reads
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    if cond then
        a.b = {}
    end
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_parent_invalidation()
    -- Assignment to 'a' invalidates 'a.b'
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    a = other_table     -- writes 'a', invalidates a.b
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_lhs_reads_parent()
    -- a.b.c = 1 reads a.b, writes a.b.c
    -- So optimizing "a.b" should work across all these
    local a_b = a.b -- opt by oLua
    a_b.c = 1
    a_b.d = 2
    a_b.e = 3
end

function test_func_arg_child()
    -- func1(a.b.c) writes a.b.c and all parents (a.b, a)
    local a_b = a.b -- opt by oLua
    local x = a_b.d
    local y = a_b.e
    func1(a_b.c)       -- invalidates a_b (parent of arg)
    local z = a_b.f
    local w = a_b.g
end

function test_no_optimization_needed()
    -- Only 1 read: not worth optimizing
    local x = a.b.c
    a.b = {}
    local y = a.b.d
end
