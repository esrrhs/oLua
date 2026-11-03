function test_arith()
    local x = 2 + 3 * 4
    local y = 1 + 2
    local z = (5 - 1) * 2
end

function test_rel_bool()
    local a = 1 < 2
    local b = not false
    local c = true and false
    local d = nil or 4
end

function test_dce_false()
    local keep = 1
    if false then
        print("dead")
    else
        print("alive")
    end
    return keep
end

function test_dce_true()
    if true then
        print("live")
    else
        print("dead")
    end
end

function test_dce_true_only()
    if true then
        local x = 1
    end
end
