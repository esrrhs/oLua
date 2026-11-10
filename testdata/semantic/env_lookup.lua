local hits = 0
local env = setmetatable({}, {__index=function(_, key)
    hits = hits + 1
    return function(x) return x + hits end
end})
local function f()
    local _ENV = env
    local x = tonumber(1)
    local y = tonumber(2)
    return x, y
end
local x, y = f()
return x, y, hits
