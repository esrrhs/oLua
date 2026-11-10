local calls = 0
local function iter() calls = calls + 1; return nil end
local function f(a)
    for x in iter do
        local y = a.b.c
        local z = a.b.c
    end
    return calls
end
return f(nil)
