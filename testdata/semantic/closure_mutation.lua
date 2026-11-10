local function f(a)
    local function change() a.b.c=20 end
    local x=a.b.c
    change()
    local y=a.b.c
    return x,y
end
return f({b={c=2}})
