local function f()
    local x = 7
    local get
    if true then
        local x = 9
        get = function() return x end
    end
    return x, get()
end
return f()
