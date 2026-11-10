local a={x=9}
local function f()
    local a={}
    a.x=a.x or 3
    return a.x
end
return f(),a.x
