-- Test advanced table access optimization scenarios

function test_basic_reads()
    -- Basic consecutive reads (should optimize a.b)
    local x = a.b.c
    local y = a.b.d
    local z = a.b.e
end

function test_write_interruption()
    -- Reads interrupted by direct write
    local x = a.b.c
    local y = a.b.d
    a.b = {}           -- write: invalidates a.b cache
    local z = a.b.e
    local w = a.b.f
end

function test_func_call_invalidation()
    -- Function call with ancestor argument invalidates
    local x = a.b.c
    local y = a.b.d
    func1(a)           -- passes 'a' to function, invalidates a.b
    local z = a.b.e
    local w = a.b.f
end

function test_method_call_invalidation()
    -- Method call on target invalidates it
    local x = a.b.c
    a.b:doSomething()  -- a.b is self, written
    local y = a.b.d
    local z = a.b.e
end

function test_nested_scope()
    -- Reads inside if block
    if cond then
        local x = a.b.c
        local y = a.b.d
        local z = a.b.e
    end
end

function test_write_in_branch()
    -- Write inside a branch invalidates outer reads
    local x = a.b.c
    local y = a.b.d
    if cond then
        a.b = {}
    end
    local z = a.b.e
    local w = a.b.f
end

function test_parent_invalidation()
    -- Assignment to 'a' invalidates 'a.b'
    local x = a.b.c
    local y = a.b.d
    a = other_table     -- writes 'a', invalidates a.b
    local z = a.b.e
    local w = a.b.f
end

function test_lhs_reads_parent()
    -- a.b.c = 1 reads a.b, writes a.b.c
    -- So optimizing "a.b" should work across all these
    a.b.c = 1
    a.b.d = 2
    a.b.e = 3
end

function test_func_arg_child()
    -- func1(a.b.c) writes a.b.c and all parents (a.b, a)
    local x = a.b.d
    local y = a.b.e
    func1(a.b.c)       -- invalidates a.b (parent of arg)
    local z = a.b.f
    local w = a.b.g
end

function test_no_optimization_needed()
    -- Only 1 read: not worth optimizing
    local x = a.b.c
    a.b = {}
    local y = a.b.d
end
