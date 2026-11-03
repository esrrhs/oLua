function test_break_skip()
    for i = 1, 10 do
        local x = a.b.c
        local y = a.b.d
        if x then
            break
        end
    end
end

function test_write_skip()
    for i = 1, 10 do
        local x = a.b.c
        local y = a.b.d
        a.b = {}
    end
end

function test_impure_skip()
    for i = 1, 10 do
        local x = a.b.c
        local y = a.b.d
        foo(a)
    end
end

function test_iter_binder_skip()
    for i = 1, 10 do
        local x = t[i].a
        local y = t[i].b
    end
end

function test_below_threshold()
    for i = 1, 10 do
        local x = a.b.c + i
    end
end

function test_nested_func_excluded()
    for i = 1, 10 do
        local function inner()
            local x = a.b.c
            local y = a.b.d
        end
        local z = i
    end
end
