-- Test semantic correctness with complex patterns

function test_mixed_rw_in_loop()
    -- Multiple reads between writes, inside a loop
    for i = 1, 10 do
        config.settings.width = i * 10
        config.settings.height = i * 20
        config.settings.depth = i * 30
        local vol = config.settings.width * config.settings.height * config.settings.depth
        print(vol)
    end
end

function test_chained_method()
    -- a.b:method() returns should not be confused with reads
    local result = a.b:getData()
    local x = a.b.name
    local y = a.b.value
end

function test_multireturn()
    -- Multiple return values from function
    local x, y = a.b.c, a.b.d
    local z = a.b.e
end

function test_concat_and_ops()
    -- Table access in concatenation and operations
    local s = a.b.name .. " " .. a.b.title
    local n = a.b.x + a.b.y * 2
    print(s, n)
end

function test_table_constructor_value()
    -- Table access as values in table constructor
    local t = {
        name = a.b.name,
        value = a.b.value,
        extra = a.b.extra,
    }
end
