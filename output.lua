local a = b

function aa()
    local a = New()
    a.b = 1
    a.c = 2
    a.d = New()
local a_d = a.d
    a_d.c = a.c + (a_d.e or 0)
    a_d.c.e = a.c + a_d.c
    a_d.d = New()
local a_d_d = a_d.d
    a_d_d.x = 1
    a_d_d.y = 2
    a_d_d.z = 3
    a[1] = New()
local a_1_ = a[1]
    a_1_.a["d"] = New()
local a_1__a__d__ = a_1_.a["d"]
    a_1__a__d__.d = 123
end
