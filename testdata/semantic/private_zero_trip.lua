local function f()
    local a={b={c=2}}
    local sum=0
    for i=1,0 do
        sum=sum+a.b.c
        sum=sum+a.b.c
    end
    return sum
end
return f()
