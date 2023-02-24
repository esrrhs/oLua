function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local b = 4

        local a  = {['b'] =  {}, ['c'] =  2, [3] =  3, [b] =  4} -- opt by oLua
        a.b.c = 5

    end

    print(os.clock() - begin)

end

test()
