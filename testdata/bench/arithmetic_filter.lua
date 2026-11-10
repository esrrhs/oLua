-- Constant arithmetic and dead configuration branches in a numeric kernel.
return function()
    local sum=0
    for i=1,200000 do
        if true then
            sum=sum+i*(2+3*4)
        else
            sum=sum-999
        end
    end
    return sum
end
