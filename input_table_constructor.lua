function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local b = 4

        local a = {}
        a.b = {}
        a["c"] = 2
        a[3] = 3
        a[b] = 4
        a.b.c = 5

    end

    print(os.clock() - begin)

end

test()
