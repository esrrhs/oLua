local a = b

function aa()
    local a = New()
    a.b = New()
    local a_b = a.b
    a_b.c = 1
    if a_b.c then
        local b = New()
        b.c = New()
        local b_c = b.c
        b_c.d = 1
    else
        local b = New()
        b.c = New()
        local b_c = b.c
        b_c.d = 2
    end
end
