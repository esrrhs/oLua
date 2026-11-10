function test_arith()
    local x = (14) -- opt by oLua
    local y = (3) -- opt by oLua
    local z = (8) -- opt by oLua
end

function test_rel_bool()
    local a = (true) -- opt by oLua
    local b = (true) -- opt by oLua
    local c = (false) -- opt by oLua
    local d = (4) -- opt by oLua
end

function test_dce_false()
    local keep = 1
    do -- opt by oLua
        print("alive")
    end
    return keep
end

function test_dce_true()
    do -- opt by oLua
        print("live")
    end
end

function test_dce_true_only()
    do -- opt by oLua
        local x = 1
    end
end
