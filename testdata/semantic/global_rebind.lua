local old = tonumber
local function f()
    local x = tonumber("1")
    tonumber = function() return 100 end
    local y = tonumber("2")
    return x, y
end
local x,y=f()
tonumber=old
return x,y
