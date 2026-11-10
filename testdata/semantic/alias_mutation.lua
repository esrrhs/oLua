local function f(a)
    local other = a.b
    local x = a.b.c
    other.c = 8
    local y = a.b.c
    return x, y
end
return f({b={c=1}})
