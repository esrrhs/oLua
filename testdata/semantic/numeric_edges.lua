local function f()
    local a=08 + 1
    local b=9223372036854775807 + 1
    local c=(-7) // 3
    local d=(-7) % 3
    local e=9007199254740993 == 9007199254740992.0
    local f=-0.0 + -0.0
    local g=1 / f
    return a,b,c,d,e,f,g
end
return f()
