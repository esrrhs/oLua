local function f()
    local a={text="a\n'b\\c"}
    a["quote'"]="x\t7"
    return a.text, a["quote'"]
end
return f()
