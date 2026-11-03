function test_if_zero_skip()
    if 0 then
        print("truthy in lua")
    end
end

function test_div0_skip()
    local x = 1 / 0
    local y = 1 // 0
    local z = 1 % 0
end

function test_goto_skip()
    if false then
        goto skip
        print(1)
        ::skip::
        print(2)
    end
end

function test_break_skip()
    for i = 1, 3 do
        if true then
            break
        end
    end
end

function test_label_skip()
    if false then
        ::lab::
        print(1)
    end
end

function test_nonconst_skip()
    local x = a + 1
    local y = 1 + b
end
