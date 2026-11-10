function math_hot()
    local a = math.abs(-1) + math.floor(1.5) + math.max(1, 2)
    local b = math.abs(-2) + math.floor(2.5) + math.max(3, 4)
    local c = table.concat({"a", "b"}, ",")
    local d = table.concat({"c", "d"}, ",")
    return a, b, c, d, os.clock(), os.clock()
end

math_hot()
