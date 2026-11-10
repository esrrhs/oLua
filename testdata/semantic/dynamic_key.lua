local function f(a, k)
    local x = a[k].c
    local y = a[k].c
    return x, y
end
return f({b={c=7}}, "b")
