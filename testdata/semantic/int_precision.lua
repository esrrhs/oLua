local function f()
    local a = 9007199254740993 == 9007199254740992
    local b = 9223372036854775807 // 3
    local c = 9223372036854775807 % 3
    return a, b, c
end
return f()
