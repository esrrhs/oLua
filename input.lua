local a = b

function aa()
    local a = New()
    a.b = 1
    a.c = 2
    a.d = New()
    a.d.c = a.c + (a.d.e or 0)
    a.d.c.e = a.c + a.d.c
    a.d.d = New()
    a.d.d.x = 1
    a.d.d.y = 2
    a.d.d.z = 3
    a[1] = New()
    a[1].a["d"] = New()
    a[1].a["d"].d = 123
end
