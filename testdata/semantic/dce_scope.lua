local function f()
    local x = 7
    if true then
        local x = 9
    end
    return x
end
return f()
