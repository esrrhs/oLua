local function f(a)
    print("first")
    local x=a.b.c
    print("middle",x)
    local y=a.b.c
    print("last",y)
    return x,y,nil,false
end
return f({b={c=5}})
