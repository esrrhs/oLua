-- Test constructor merging with unrelated statements interleaved,
-- nested merging, same-key overwrite, and dynamic-key blocking.

function test_interleave()
    local a = {['b']=1, ['c']=2, ['d']=3} -- opt by oLua
    local x = 5
    print("hello")
    local y = x + 1
end

function test_nested()
    local a = {['d']={['e']=1, ['f']=2, ['g']=3}} -- opt by oLua
end

function test_same_key()
    local a = {['x']=3, ['y']=2} -- opt by oLua
end

function test_read_blocks()
    local a = {['b']=1} -- opt by oLua
    local y = a.c
    a.d = 2
end

function test_dynamic_key_blocks()
    local a = {['b']=1} -- opt by oLua
    a[y] = 2
    a.c = 3
end

function test_pure_call_allowed()
    local a = {['b']=1, ['c']=2} -- opt by oLua
    print("ok")
end
