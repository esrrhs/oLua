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
        a.b.c = 7

    end

    print(os.clock() - begin)

end

test()