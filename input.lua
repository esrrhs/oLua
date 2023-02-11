local a = b

function aa()
    local a = New()
    a.b = New()
    a.b.c = 1
    if a.b.c then
        local b = New()
        b.c = New()
        b.c.d = 1
    else
        local b = New()
        b.c = New()
        b.c.d = 2
    end
end
