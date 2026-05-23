-- Test variable name conflict detection

function test_name_conflict()
    -- User already has a variable called 'a_b'
    local a_b = 123
    local a_b_1 = a.b -- opt by oLua
    local x = a_b_1.c
    local y = a_b_1.d
    local z = a_b_1.e
    print(a_b)
end

function test_no_conflict()
    -- No existing variable named 'a_b', should use it directly
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    local z = a_b.e
end
