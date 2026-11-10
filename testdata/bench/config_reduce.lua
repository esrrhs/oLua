-- A configuration-driven weighted reduction (200,000 records).
return function()
    local cfg = {weight = {x=2, y=3}, offset = {x=5, y=7}}
    local sum=0
    for i=1,200000 do
        sum=sum+(i+cfg.offset.x)*cfg.weight.x
        sum=sum+(i+cfg.offset.y)*cfg.weight.y
    end
    return sum
end
