local hits = 0
local a = setmetatable({}, {__index = function()
    hits = hits + 1
    return {c = hits}
end})
local function f(a)
    local sum = 0
    for i = 1, 3 do
        sum = sum + a.b.c
        sum = sum + a.b.c
    end
    return sum
end
return f(a), hits
