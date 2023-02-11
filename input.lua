local a = b

function aa()
    local a = New()
    a.b = New()
    a.b.c = 1
    if a.b.c then
        local b = New()
        b.c = New()
        b.c.d = 1
    elseif a.b.d then
        local b = New()
        b.c = New()
        b.c.d = 2
        b.c.e = 2
    else
        local b = New()
        b.c = New()
        b.c.d = 3
        b.c.e = 3
        b.c.f = 3
    end
    a.b.d = 1
end
