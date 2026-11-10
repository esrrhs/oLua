local hits=0
local function f()
    local a={}
    if true then
        setmetatable(a,{__newindex=function(t,k,v) hits=hits+1; rawset(t,k,v+1) end})
    end
    a.x=7
    return a.x
end
return f(),hits
