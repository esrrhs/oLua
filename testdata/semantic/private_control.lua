local function f()
    local a={b={c=2,d=3}}
    local sum=0
    if false then
        sum=sum+a.b.c
    else
        sum=sum+a.b.d
    end
    do
        sum=sum+a.b.c
    end
    return sum+a.b.d
end
return f()
