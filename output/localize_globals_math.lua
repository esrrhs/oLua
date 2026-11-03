function math_hot()
    local abs, floor, max, concat, clock = math.abs, math.floor, math.max, table.concat, os.clock -- opt by oLua
    local a = abs(-1) + floor(1.5) + max(1, 2)
    local b = abs(-2) + floor(2.5) + max(3, 4)
    local c = concat({"a", "b"}, ",")
    local d = concat({"c", "d"}, ",")
    return a, b, c, d, clock(), clock()
end

math_hot()
