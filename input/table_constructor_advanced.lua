-- Test constructor merging with unrelated statements interleaved,
-- nested merging, same-key overwrite, and dynamic-key blocking.

function test_interleave()
    local a = {}
    local x = 5
    a.b = 1
    print("hello")
    a.c = 2
    local y = x + 1
    a.d = 3
end

function test_nested()
    local a = {}
    a.d = { e = 1 }
    a.d.f = 2
    a.d.g = 3
end

function test_same_key()
    local a = {}
    a.x = 1
    a.y = 2
    a.x = 3
end

function test_read_blocks()
    local a = {}
    a.b = 1
    local y = a.c
    a.d = 2
end

function test_dynamic_key_blocks()
    local a = {}
    a.b = 1
    a[y] = 2
    a.c = 3
end

function test_pure_call_allowed()
    local a = {}
    a.b = 1
    print("ok")
    a.c = 2
end
