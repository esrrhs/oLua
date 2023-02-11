local a = b

function aa()
    local a = New()
    a.b = New()
    local a_b = a.b -- opt by lua2lua
    a_b.c = 1
    if a_b.c then
        local b = New()
        b.c = New()
        b.c.d = 1
    elseif a_b.d then
        local b = New()
        b.c = New()
        local b_c = b.c -- opt by lua2lua
        b_c.d = 2
        b_c.e = 2
    else
        local b = New()
        b.c = New()
        local b_c = b.c -- opt by lua2lua
        b_c.d = 3
        b_c.e = 3
        b_c.f = 3
    end
    a_b.d = 1
end
