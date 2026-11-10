local old = math.abs
local function change() math.abs = function(x) return 99 end end
local function f()
    local x = math.abs(-1)
    change()
    local y = math.abs(-2)
    return x, y
end
local x, y = f()
math.abs = old
return x, y
