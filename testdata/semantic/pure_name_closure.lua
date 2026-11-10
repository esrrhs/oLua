local function f(a)
    local print = function() a.b.c = 42 end
    local x = a.b.c
    print()
    local y = a.b.c
    return x, y
end
return f({b = {c = 1}})
