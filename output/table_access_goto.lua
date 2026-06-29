function test_label_breaks_group()
    -- label 之前和之后的读应该被分隔
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    ::point::
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_goto_breaks_group()
    local a_b = a.b -- opt by oLua
    local x = a_b.c
    local y = a_b.d
    if cond then goto skip end
    a.b = {}
    ::skip::
    a_b = a.b -- opt by oLua
    local z = a_b.e
    local w = a_b.f
end

function test_break_in_loop()
    -- break 是 Goto IsBreak=true，循环内的 break 影响 break 之后
    for i = 1, 10 do
        local a_b = a.b -- opt by oLua
        local x = a_b.c
        local y = a_b.d
        if cond then break end
        local z = a.b.e
    end
end

function test_continue()
    -- 用户上次代码中有 ::continue::
    for i = 1, 10 do
        local x = a.b.c
        if cond then goto continue end
        local y = a.b.d
        ::continue::
        local z = a.b.e
    end
end
