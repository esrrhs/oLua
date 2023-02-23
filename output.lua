function test()

    local begin = os.clock()

    for i = 1, 1000000 do

        local a = {}
        a.b = {}
        local a_b = a.b -- opt by lua2lua
        a_b.data = "a"
        a_b.c = {}
        local a_b_c = a_b.c -- opt by lua2lua
        a_b_c.data = "b"
        a_b_c.d = {}
        local a_b_c_d = a_b_c.d -- opt by lua2lua
        a_b_c_d.data = "c"
        a_b_c_d.e = {}
        local a_b_c_d_e = a_b_c_d.e -- opt by lua2lua
        a_b_c_d_e.data = "d"
        a_b_c_d_e.f = {}
        local a_b_c_d_e_f = a_b_c_d_e.f -- opt by lua2lua
        a_b_c_d_e_f.data = "e"
        a_b_c_d_e_f.g = {}
        local a_b_c_d_e_f_g = a_b_c_d_e_f.g -- opt by lua2lua
        a_b_c_d_e_f_g.data = "f"
        a_b_c_d_e_f_g.h = {}
        local a_b_c_d_e_f_g_h = a_b_c_d_e_f_g.h -- opt by lua2lua
        a_b_c_d_e_f_g_h.data = "g"
        a_b_c_d_e_f_g_h.i = {}
        local a_b_c_d_e_f_g_h_i = a_b_c_d_e_f_g_h.i -- opt by lua2lua
        a_b_c_d_e_f_g_h_i.data = "h"
        a_b_c_d_e_f_g_h_i.j = {}
        local a_b_c_d_e_f_g_h_i_j = a_b_c_d_e_f_g_h_i.j -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j.data = "i"
        a_b_c_d_e_f_g_h_i_j.k = {}
        local a_b_c_d_e_f_g_h_i_j_k = a_b_c_d_e_f_g_h_i_j.k -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k.data = "j"
        a_b_c_d_e_f_g_h_i_j_k.l = {}
        local a_b_c_d_e_f_g_h_i_j_k_l = a_b_c_d_e_f_g_h_i_j_k.l -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l.data = "k"
        a_b_c_d_e_f_g_h_i_j_k_l.m = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m = a_b_c_d_e_f_g_h_i_j_k_l.m -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m.data = "l"
        a_b_c_d_e_f_g_h_i_j_k_l_m.n = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n = a_b_c_d_e_f_g_h_i_j_k_l_m.n -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n.data = "m"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n.o = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o = a_b_c_d_e_f_g_h_i_j_k_l_m_n.o -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o.data = "n"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o.p = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o.p -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p.data = "o"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p.q = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p.q -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q.data = "p"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q.r = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q.r -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r.data = "q"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r.s = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r.s -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s.data = "r"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s.t = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s.t -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t.data = "s"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t.u = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t.u -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u.data = "t"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u.v = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u.v -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v.data = "u"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v.w = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v.w -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w.data = "v"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w.x = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w.x -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x.data = "w"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x.y = {}
        local a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x_y = a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x.y -- opt by lua2lua
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x_y.data = "x"
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x_y.z = {}
        a_b_c_d_e_f_g_h_i_j_k_l_m_n_o_p_q_r_s_t_u_v_w_x_y.z.data = "y"

    end

    print(os.clock() - begin)

end

test()
