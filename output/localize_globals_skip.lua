-- already local print: should not localize print
function with_local_print()
    local print = error
    print(1)
    print(2)
    print(3)
end

-- global assign to print: skip print
function with_print_assign()
    print = error
    print(1)
    print(2)
    print(3)
end

-- _ENV rebind: skip whole function
function with_env()
    _ENV = {}
    print(1)
    print(2)
    print(3)
end

-- method calls are not string.format; only two string.format -> hoist format
function with_methods()
    local format = string.format -- opt by oLua
    o:format()
    o:format()
    o:format()
    format("%s", 1)
    format("%s", 2)
end

-- nested function body excluded from outer counts (outer has only 1 print)
function outer_only_one()
    print(1)
    local function inner()
        local print = print -- opt by oLua
        print(2)
        print(3)
        print(4)
    end
    inner()
end

-- conflict alias: local format exists, string.format should get format_1
function conflict_format()
    local format_1 = string.format -- opt by oLua
    local format = 1
    use(format_1("%d", 1))
    use(format_1("%d", 2))
    use(format)
end

with_local_print()
with_print_assign()
with_env()
with_methods()
outer_only_one()
conflict_format()
