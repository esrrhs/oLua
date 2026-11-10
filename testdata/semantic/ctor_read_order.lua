local function f()
    local x=1
    local a={}
    x=2
    a.value=x
    return a.value
end
return f()
