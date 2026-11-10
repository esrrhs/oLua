local function f()
    local a = {b = {c = 2, d = 3}}
    local sum = 0
    for i = 1, 10 do
        sum = sum + a.b.c
        sum = sum + a.b.d
    end
    return sum
end
return f()
