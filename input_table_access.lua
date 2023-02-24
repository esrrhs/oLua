function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local a = {}
        a.b = {}
        a.b.data = "a"
        a.b.c = {}
        a.b.c.data = "b"
        a.b.c.d = {}
        a.b.c.d.data = "c"
        a.b.c.d.e = {}
        a.b.c.d.e.data = "d"
        a.b.c.d.e.f = {}
        a.b.c.d.e.f.data = "e"
        a.b.c.d.e.f.g = {}
        a.b.c.d.e.f.g.data = "f"
        a.b.c.d.e.f.g.h = {}
        a.b.c.d.e.f.g.h.data = "g"
        a.b.c.d.e.f.g.h.i = {}
        a.b.c.d.e.f.g.h.i.data = "h"
        a.b.c.d.e.f.g.h.i.j = {}
        a.b.c.d.e.f.g.h.i.j.data = "i"
        a.b.c.d.e.f.g.h.i.j.k = {}
        a.b.c.d.e.f.g.h.i.j.k.data = "j"
        a.b.c.d.e.f.g.h.i.j.k.l = {}
        a.b.c.d.e.f.g.h.i.j.k.l.data = "k"
        a.b.c.d.e.f.g.h.i.j.k.l.m = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.data = "l"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.data = "m"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.data = "n"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.data = "o"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.data = "p"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.data = "q"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.data = "r"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.data = "s"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.data = "t"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.data = "u"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.data = "v"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.data = "w"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.data = "x"
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z = {}
        a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u.v.w.x.y.z.data = "y"

    end

    print(os.clock() - begin)

end

test()
