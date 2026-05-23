-- Test semantic correctness with complex patterns

function test_mixed_rw_in_loop()
    -- Multiple reads between writes, inside a loop
    for i = 1, 10 do
        local config_settings = config.settings -- opt by oLua
        config_settings.width = i * 10
        config_settings.height = i * 20
        config_settings.depth = i * 30
        local vol = config_settings.width * config_settings.height * config_settings.depth
        print(vol)
    end
end

function test_chained_method()
    -- a.b:method() returns should not be confused with reads
    local a_b = a.b -- opt by oLua
    local result = a_b:getData()
    local x = a_b.name
    local y = a_b.value
end

function test_multireturn()
    -- Multiple return values from function
    local a_b = a.b -- opt by oLua
    local x, y = a_b.c, a_b.d
    local z = a_b.e
end

function test_concat_and_ops()
    -- Table access in concatenation and operations
    local a_b = a.b -- opt by oLua
    local s = a_b.name .. " " .. a_b.title
    local n = a_b.x + a_b.y * 2
    print(s, n)
end

function test_table_constructor_value()
    -- Table access as values in table constructor
    local a_b = a.b -- opt by oLua
    local t = {
        name = a_b.name,
        value = a_b.value,
        extra = a_b.extra,
    }
end
