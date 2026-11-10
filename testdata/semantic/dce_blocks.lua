local function f()
local x=4
if false then
  error("dead")
elseif true then
 do
  local x=10
  local function g() return x end
  return g(), x
 end
else
 error("dead too")
end
return x
end
return f()
