import json,sys,datetime as t
d=json.load(open(sys.argv[1]));f=d["fields"];n=d.get("names",{})
print("sourceURL:",d["self"].split("/rest/api/")[0]+"/browse/"+d["key"])
print("sourceUpdatedAt:",t.datetime.strptime(f["updated"],"%Y-%m-%dT%H:%M:%S.%f%z").isoformat(timespec="milliseconds"))
print("title:",d["key"],f["summary"],"| status:",f["status"]["name"])
print("parent:",(f.get("parent") or {}).get("key"),"subtasks:",[s["key"] for s in f.get("subtasks",[])])
for l in f.get("issuelinks",[]):
  s="inwardIssue" if "inwardIssue" in l else "outwardIssue"
  if l["type"][s[:-5]]=="is blocked by":print("blocked by:",l[s]["key"])
for k,v in n.items():
  if v=="Epic Link" or "acceptance" in v.lower():print(v+":",f.get(k))
print(f.get("description"))
for c in f["comment"]["comments"]:print(c["created"],c["author"]["displayName"],c["body"])
