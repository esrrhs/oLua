-- Construct fixed record defaults, then consume them (100,000 records).
return function()
    local sum=0
    for i=1,100000 do
        local record={}
        record.x=2
        record.y=3
        record.meta={weight=5}
        sum=sum+record.x+record.y+record.meta.weight
    end
    return sum
end
