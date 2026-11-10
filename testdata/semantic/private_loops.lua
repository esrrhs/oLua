local function f()
    local a={b={c=2,d=3}}
    local n=0
    local sum=0
    while n<3 do
        n=n+1
        sum=sum+a.b.c
        sum=sum+a.b.d
    end
    repeat
        sum=sum+a.b.c
        sum=sum+a.b.d
        n=n-1
    until n==0
    return sum
end
return f()
