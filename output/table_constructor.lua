local function tmp()
    return 1
end

function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local b = 4

        local a = {a=1, 2, b={}, c=3, [3]=4, [b]=5, d={e=6,f=7,[1]=8}, e=os.time() or 0, f={1,2,3}, g='str'..' '..i, h=(2+3)*2-1, i=tmp()} -- opt by oLua
        a.b.c = 9

    end

    print(os.clock() - begin)

end

test()
