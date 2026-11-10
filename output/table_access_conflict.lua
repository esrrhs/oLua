-- Test variable name conflict detection

function test_name_conflict()
    -- User already has a variable called 'a_b'
    local a_b = 123
    local x = a.b.c
    local y = a.b.d
    local z = a.b.e
    print(a_b)
end

function test_no_conflict()
    -- No existing variable named 'a_b', should use it directly
    local x = a.b.c
    local y = a.b.d
    local z = a.b.e
end
