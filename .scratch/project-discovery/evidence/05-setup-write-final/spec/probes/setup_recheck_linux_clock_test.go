//go:build linux
package discovery

import (
 "bytes"
 "context"
 "encoding/binary"
 "os"
 "syscall"
 "testing"

 "golang.org/x/sys/unix"
)

func TestReviewLinuxACLClockCollision(t *testing.T){
 for attempt:=0;attempt<100;attempt++{
  request,path:=setupWriteFixture(t,true)
  initialACL:=reviewLinuxACL()
  if err:=unix.Setxattr(path,"system.posix_acl_access",initialACL,0);err!=nil{t.Fatal(err)}
  before,_:=os.Stat(path)
  plan:=setupWritePlan(t,request)
  // Revoke the named user's read grant without changing UID, GID, or mode.
  revokedACL:=bytes.Clone(initialACL)
  binary.LittleEndian.PutUint16(revokedACL[14:16],0)
  var changed os.FileInfo
  ops:=defaultSetupWriteOps()
  ops.createTemp=func(dir,pattern string)(setupTempFile,error){
   f,err:=os.CreateTemp(dir,pattern)
   return &reviewLinuxTemp{f,func(){
    if err:=unix.Setxattr(path,"system.posix_acl_access",revokedACL,0);err!=nil{t.Fatal(err)}
    changed,err=os.Stat(path);if err!=nil{t.Fatal(err)}
   }},err
  }
  err:=plan.apply(context.Background(),ops)
  if changed==nil{t.Fatalf("no concurrent change: %v",err)}
  oldStat,newStat:=before.Sys().(*syscall.Stat_t),changed.Sys().(*syscall.Stat_t)
  if oldStat.Ctim==newStat.Ctim{
   afterACL:=reviewLinuxGetXattr(t,path,"system.posix_acl_access")
   t.Logf("attempt=%d beforeCtim=%+v changedCtim=%+v beforeMode=%o changedMode=%o applyErr=%v initialNamedPerm=%d revokedNamedPerm=%d finalNamedPerm=%d",attempt,oldStat.Ctim,newStat.Ctim,before.Mode(),changed.Mode(),err,binary.LittleEndian.Uint16(initialACL[14:16]),binary.LittleEndian.Uint16(revokedACL[14:16]),binary.LittleEndian.Uint16(afterACL[14:16]))
   if err==nil&&!bytes.Equal(afterACL,revokedACL){t.Fatal("same-tick ACL revocation was overwritten by replacement")}
  }
 }
}
