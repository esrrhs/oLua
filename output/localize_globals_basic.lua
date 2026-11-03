function hot()
    local print, format = print, string.format -- opt by oLua
    print(1)
    print(2)
    print(format("%d", 3))
    print(format("%d", 4))
end

hot()
