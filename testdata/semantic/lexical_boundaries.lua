local function f(a)
    local text=[=[a.b.c 1+2
1+2 end if true then]=]
    local x = a - 1; local y = -1
    local z = a*1+2; local k = 1+2
    local n = 11+2; local m = 1+2
    return text,x,y,z,k,n,m
end
return f(7)
