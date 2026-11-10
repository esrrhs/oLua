-- Edge cases for table constructor optimization.

-- 1) RHS reads target: should stop absorbing at that assignment.
function test_rhs_reads_target()
    local a = {['b']=1} -- opt by oLua
    a.c = a.b + 1   -- stops here: rhs reads a
    a.d = 2
end

-- 2) RHS calls a function that may mutate target: should stop.
function test_rhs_calls_impure_on_target()
    local a = {['b']=1} -- opt by oLua
    a.c = f(a)      -- stops here: f(a) may write a.*
    a.d = 2
end

-- 3) Loop interleaved: conservative stop.
function test_loop_interleaves()
    local a = {['b']=1} -- opt by oLua
    for i = 1, 3 do end
    a.c = 2
end

-- 4) goto / label interleaved: conservative stop.
function test_goto_interleaves()
    local a = {['b']=1} -- opt by oLua
    goto L
    ::L::
    a.c = 2
end

-- 5) Non-local target: s.o = {} should still merge.
function test_non_local_target()
    s.o = {}
    s.o.k = 1
    s.o.m = 2
end

-- 6) Inside an if branch.
function test_inside_if()
    if cond then
        local a = {['b']=1, ['c']=2} -- opt by oLua
    end
end

-- 7) Multiple independent tables: both should merge across rounds.
function test_multiple_tables()
    local a = {}
    local b = {}
    a.x = 1
    b.y = 1
    a.z = 2
    b.w = 2
end

-- 8) Sub-table clobbered by leaf value (same-key overwrite crossing types).
function test_subtable_clobbered()
    local a = {['d']=5} -- opt by oLua
end

-- 9) Array positions + named keys mixed in initializer.
function test_array_plus_named()
    local a = { 1, 2 }
    a.b = 3
    a.c = 4
end

-- 10) Multi-line RHS constructor: all lines must be removed.
function test_multiline_rhs()
    local a = {}
    a.f = {
        1, 2, 3
    }
    a.g = 5
end
