local calls = 0
local function hit() calls = calls + 1; return calls end
local function f()
    local a = {x = hit()}
    a.x = hit()
    return a.x
end
return f(), calls
