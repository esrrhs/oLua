local function f()
    local a={text="{"}
    a.value="}"
    return a.text,a.value
end
return f()
