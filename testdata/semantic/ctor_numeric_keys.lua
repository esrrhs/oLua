local function f()
    local a={[1]=1,[01]=2}
    a[1]=3
    return a[1]
end
local function g()
    local a={}
    a[1]=3
    a[01]=4
    a[1]=5
    return a[1]
end
return f(),g()
