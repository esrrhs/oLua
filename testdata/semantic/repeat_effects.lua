local hits = 0
local a = setmetatable({}, {__index=function() hits=hits+1; return hits end})
local function f(a)
    local n = 0
    repeat
        n = n + a.x
        n = n + a.x
    until n > 10
    return n
end
return f(a), hits
