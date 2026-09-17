//go:build linux

package discovery

import (
 "bytes"
 "context"
 "encoding/binary"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "syscall"
 "testing"

 "golang.org/x/sys/unix"
)

// Independent fixture encoding follows Linux v6.10 UAPI posix_acl_xattr.h and
// posix_acl.h, not a production helper. Version 2 and five eight-byte entries.
// https://raw.githubusercontent.com/torvalds/linux/v6.10/include/uapi/linux/posix_acl_xattr.h
func reviewLinuxACL()[]byte{
 data:=make([]byte,44)
 binary.LittleEndian.PutUint32(data,2)
 entries:=[]struct{tag,perm uint16;id uint32}{{1,6,0xffffffff},{2,4,1234},{4,4,0xffffffff},{16,4,0xffffffff},{32,0,0xffffffff}}
 for i,e:=range entries{at:=data[4+8*i:];binary.LittleEndian.PutUint16(at,e.tag);binary.LittleEndian.PutUint16(at[2:],e.perm);binary.LittleEndian.PutUint32(at[4:],e.id)}
 return data
}
func reviewLinuxGetXattr(t *testing.T,path,name string)[]byte{
 t.Helper();size,err:=unix.Getxattr(path,name,nil);if err!=nil{t.Fatal(err)};value:=make([]byte,size);n,err:=unix.Getxattr(path,name,value);if err!=nil{t.Fatal(err)};return value[:n]
}
func reviewLinuxSetACL(t *testing.T,path,name string){t.Helper();if err:=unix.Setxattr(path,name,reviewLinuxACL(),0);err!=nil{t.Fatalf("kernel rejected independent ACL fixture: %v",err)}}

func TestReviewLinuxPreservesOwnerGroupACL(t *testing.T){
 request,path:=setupWriteFixture(t,true)
 if err:=os.Chown(path,1001,1002);err!=nil{t.Fatal(err)}
 reviewLinuxSetACL(t,path,"system.posix_acl_access")
 acl:=reviewLinuxGetXattr(t,path,"system.posix_acl_access")
 before,_:=os.Stat(path)
 plan:=setupWritePlan(t,request)
 if err:=plan.Apply(context.Background());err!=nil{t.Fatal(err)}
 after,_:=os.Stat(path);st:=after.Sys().(*syscall.Stat_t)
 if st.Uid!=1001||st.Gid!=1002||after.Mode()!=before.Mode()||!bytes.Equal(acl,reviewLinuxGetXattr(t,path,"system.posix_acl_access")){t.Fatalf("source permissions changed: %#v",st)}
 t.Logf("retained uid=%d gid=%d mode=%v and %d-byte POSIX ACL",st.Uid,st.Gid,after.Mode(),len(acl))
}
func TestReviewLinuxClearsInheritedACL(t *testing.T){
 request,path:=setupWriteFixture(t,true)
 reviewLinuxSetACL(t,filepath.Dir(path),"system.posix_acl_default")
 before,_:=os.Stat(path)
 plan:=setupWritePlan(t,request)
 if err:=plan.Apply(context.Background());err!=nil{t.Fatal(err)}
 after,_:=os.Stat(path)
 if _,err:=unix.Getxattr(path,"system.posix_acl_access",nil);err!=unix.ENODATA{t.Fatalf("inherited access grant remains: %v",err)}
 if after.Mode()!=before.Mode(){t.Fatal("mode changed clearing inherited ACL")}
}

type reviewLinuxTemp struct{*os.File; change func()}
func(f *reviewLinuxTemp)Sync()error{f.change();return f.File.Sync()}
func TestReviewLinuxDetectsConcurrentPermissions(t *testing.T){
 for _,kind:=range []string{"uid","gid","acl"}{
  for _,during:=range []bool{false,true}{
   t.Run(fmt.Sprintf("%s-during-write=%t",kind,during),func(t *testing.T){
    request,path:=setupWriteFixture(t,true)
    original:=setupWriteRead(t,path)
    plan:=setupWritePlan(t,request)
    change:=func(){
     switch kind{case "uid":if err:=os.Chown(path,1001,-1);err!=nil{t.Fatal(err)};case "gid":if err:=os.Chown(path,-1,1002);err!=nil{t.Fatal(err)};case "acl":reviewLinuxSetACL(t,path,"system.posix_acl_access")}
    }
    ops:=defaultSetupWriteOps()
    if during{ops.createTemp=func(dir,pattern string)(setupTempFile,error){f,err:=os.CreateTemp(dir,pattern);return &reviewLinuxTemp{f,change},err}}else{change()}
    err:=plan.apply(context.Background(),ops)
    if err==nil||!strings.Contains(err.Error(),"changed since setup read"){t.Fatalf("metadata change ignored: %v",err)}
    if !bytes.Equal(original,setupWriteRead(t,path)){t.Fatal("stale writer changed source")}
    after,_:=os.Stat(path);st:=after.Sys().(*syscall.Stat_t)
    if kind=="uid"&&st.Uid!=1001{t.Fatal("concurrent uid reset")};if kind=="gid"&&st.Gid!=1002{t.Fatal("concurrent gid reset")}
    if kind=="acl"&&!bytes.Equal(reviewLinuxACL(),reviewLinuxGetXattr(t,path,"system.posix_acl_access")){t.Fatal("concurrent ACL reset")}
   })
  }
 }
}
func TestReviewLinuxRefusesToStripCapabilities(t *testing.T){
 request,path:=setupWriteFixture(t,true)
 // Linux UAPI capability.h: revision 2, effective, CAP_NET_BIND_SERVICE.
 // https://raw.githubusercontent.com/torvalds/linux/v6.10/include/uapi/linux/capability.h
 capability:=make([]byte,20);binary.LittleEndian.PutUint32(capability,0x02000001);binary.LittleEndian.PutUint32(capability[4:],1<<10)
 if err:=unix.Setxattr(path,"security.capability",capability,0);err!=nil{t.Fatalf("cannot set test capability: %v",err)}
 want:=reviewLinuxGetXattr(t,path,"security.capability")
 before,_:=os.Stat(path);original:=setupWriteRead(t,path)
 plan:=setupWritePlan(t,request)
 err:=plan.Apply(context.Background())
 if err==nil||!strings.Contains(err.Error(),"security.capability"){t.Fatalf("unsupported attribute did not fail safely: %v",err)}
 after,_:=os.Stat(path)
 if !os.SameFile(before,after)||!bytes.Equal(original,setupWriteRead(t,path))||!bytes.Equal(want,reviewLinuxGetXattr(t,path,"security.capability")){t.Fatal("failed write stripped capability or changed source")}
 t.Logf("safe refusal: %v",err)
}
