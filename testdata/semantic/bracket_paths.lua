local function f()
    local a={["end"]={x=1},b={c=2}}
    local sum=0
    for i=1,3 do
        sum=sum+a["end"].x
        sum=sum+a["end"].x
        sum=sum+a["b"].c
        sum=sum+a["b"].c
    end
    return sum
end
return f()
