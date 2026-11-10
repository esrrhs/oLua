-- Representative billing workload with deterministic generated input.
-- Includes calls, data-dependent branches, tables, and a reduction. It is
-- intentionally outside the current private-table proof: no speedup promised.
return function()
    local rows={}
    for i=1,2000 do
        rows[i]={price=100+i%97,quantity=1+i%8,discount=i%5==0}
    end
    local total=0
    for batch=1,20 do
        for _,row in ipairs(rows) do
            local subtotal=row.price*row.quantity
            if row.discount then
                subtotal=math.floor(subtotal*0.9)
            end
            total=total+math.floor(subtotal*1.13)
        end
    end
    return total
end
