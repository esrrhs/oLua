local function f(a)
    while false do
        local x = a.b.c
        local y = a.b.c
    end
    return 0
end
return f(nil)
