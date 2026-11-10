local function f(a)
    local x=a.b.c
    do
        local a={b={c=9}}
        x=x+a.b.c
    end
    local y=a.b.c
    return x,y
end
return f({b={c=2}})
