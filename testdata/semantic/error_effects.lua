local a = setmetatable({}, {__index=function() print("lookup"); error("lookup failed") end})
local function f(a)
    print("before")
    local x = a.b.c
    local y = a.b.c
    return x, y
end
return f(a)
