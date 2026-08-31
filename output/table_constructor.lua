local function tmp()
    return 1
end

function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local b = 4

        local a = {['a']=1, 2, ['b']={}, ['c']=3, [3]=4} -- opt by oLua
        a[b] = 5
        a.d = {['e']=6, ['f']=7, [1]=8} -- opt by oLua
        a.e = os.time() or 0
        a.f = {
            1, 2, 3
        }
        a.g = "str" .. " " .. i
        a.h = (2 + 3) * 2 - 1
        a.i = tmp()
        a.b.c = 9

    end

    print(os.clock() - begin)

end

test()
