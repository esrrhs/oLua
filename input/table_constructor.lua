function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local b = 4

        local a = { a = 1, 2 }
        a.b = {}
        a["c"] = 3
        a[3] = 4
        a[b] = 5
        a.d = { e = 6 }
        a.d.f = 7
        a.d[1] = 8
        a.e = f() or 0
        a.f = {
            1, 2, 3
        }
        a.g = "str" .. " " .. i
        a.h = (2 + 3) * 2 - 1
        a.b.c = 9

    end

    print(os.clock() - begin)

end

test()
