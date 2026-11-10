#!/usr/bin/env python3
"""Real Lua CPU-time/size benchmark, no third-party Python packages required."""
import argparse
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import statistics
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
FLAGS = ["-opt_const_fold", "-opt_table_constructor", "-opt_localize_globals", "-opt_licm", "-opt_table_access"]
DRIVER = r"""
local before, after = assert(loadfile(arg[1]))(), assert(loadfile(arg[2]))()
local warmups, repeats, batch = tonumber(arg[3]), tonumber(arg[4]), tonumber(arg[5])
local expected = before()
assert(type(expected)=="number", "fixture must return a numeric checksum")
assert(after()==expected, "WRONG RESULT: optimized checksum")
for i=1,warmups do assert(before()==expected); assert(after()==expected) end
local function measure(fn)
    collectgarbage("collect")
    local start=os.clock()
    for i=1,batch do assert(fn()==expected, "WRONG RESULT during timing") end
    return (os.clock()-start)/batch
end
for i=1,repeats do
    local a,b
    if i%2==1 then a=measure(before); b=measure(after)
    else b=measure(after); a=measure(before) end
    io.write(string.format("%.9f %.9f\n",a,b))
end
io.stderr:write("checksum=",tostring(expected),"\n")
"""

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--lua",default=os.environ.get("OLUA_LUA","lua").split(os.pathsep)[0])
    parser.add_argument("--warmups",type=int,default=2)
    parser.add_argument("--repeats",type=int,default=7)
    parser.add_argument("--batch",type=int,default=3)
    parser.add_argument("--output",type=Path)
    args=parser.parse_args()
    if args.warmups<0 or args.repeats<2 or args.batch<1: parser.error("need warmups>=0, repeats>=2, batch>=1")
    lua=shutil.which(args.lua)
    if not lua: parser.error("Lua 5.3/5.4 executable not found")
    version=subprocess.check_output([lua,"-v"],stderr=subprocess.STDOUT,text=True).strip()
    if not ("Lua 5.3" in version or "Lua 5.4" in version): parser.error("benchmark requires Lua 5.3/5.4")
    cpu=platform.processor()
    if Path("/proc/cpuinfo").exists():
        cpu=next((s.split(":",1)[1].strip() for s in Path("/proc/cpuinfo").read_text().splitlines() if s.startswith("model name")),cpu)
    fingerprint=hashlib.sha256()
    for source in sorted(ROOT.glob("*.go")):
        fingerprint.update(source.name.encode()+b"\0"+source.read_bytes()+b"\0")
    report={"generated_at":datetime.now().astimezone().isoformat(),"optimizer_go_sources_sha256":fingerprint.hexdigest(),"environment":{"platform":platform.platform(),"cpu":cpu,"lua":version,"lua_path":lua,"go":subprocess.check_output(["go","version"],text=True).strip()},"flags":FLAGS,"warmups":args.warmups,"repeats":args.repeats,"batch":args.batch,"timer":"Lua os.clock CPU seconds per function invocation; alternating paired order; GC before each timed batch", "results":[]}
    with tempfile.TemporaryDirectory(prefix="olua-benchmark-") as directory:
        directory=Path(directory)
        binary=directory/"oLua"
        subprocess.run(["go","build","-o",str(binary),"."],cwd=ROOT,check=True)
        driver=directory/"driver.lua";driver.write_text(DRIVER)
        for fixture in sorted((ROOT/"testdata/bench").glob("*.lua")):
            optimized=directory/fixture.name
            subprocess.run([str(binary),"-input",str(fixture),"-output",str(optimized),*FLAGS],cwd=ROOT,check=True,capture_output=True,text=True,timeout=30)
            run=subprocess.run([lua,str(driver),str(fixture),str(optimized),str(args.warmups),str(args.repeats),str(args.batch)],check=True,text=True,capture_output=True,timeout=180)
            pairs=[list(map(float,line.split())) for line in run.stdout.splitlines()]
            if len(pairs)!=args.repeats: raise RuntimeError("unexpected timing output: "+run.stdout)
            def summary(samples):
                return {"median_ms":statistics.median(samples)*1000,"min_ms":min(samples)*1000,"max_ms":max(samples)*1000,"stdev_ms":statistics.stdev(samples)*1000,"samples_seconds":samples}
            before=summary([p[0] for p in pairs]);after=summary([p[1] for p in pairs])
            report["results"].append({"workload":fixture.stem,"fixture_sha256":hashlib.sha256(fixture.read_bytes()).hexdigest(),"original_bytes":fixture.stat().st_size,"optimized_bytes":optimized.stat().st_size,"original":before,"optimized":after,"median_time_ratio_original_over_optimized":before["median_ms"]/after["median_ms"],"checksum":run.stderr.strip()})
    content=json.dumps(report,ensure_ascii=False,indent=2)+"\n"
    if args.output: args.output.write_text(content)
    print(content,end="")

if __name__=="__main__": main()
