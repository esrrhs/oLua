local function f()
    local a={}
    a.x=2
    a.y={z=3}
    return a.x+a.y.z
end
return f()
