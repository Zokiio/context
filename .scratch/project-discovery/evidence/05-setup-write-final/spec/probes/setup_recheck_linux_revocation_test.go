//go:build linux
package discovery

import (
 "bytes"
 "context"
 "encoding/binary"
 "os"
 "strings"
 "syscall"
 "testing"

 "golang.org/x/sys/unix"
)

// Preserve the earlier red probe separately. This stricter recheck asserts
// rejection, no inode/content replacement, and the revoked ACL on every attempt.
func TestReviewLinuxRevocationSurvivesNaturalClockCollision(t *testing.T) {
 for _,during:=range []bool{false,true} {
  label:="before apply";if during{label="during sync"}
  t.Run(label,func(t *testing.T){
   collisions:=0
   for attempt:=0;attempt<100;attempt++ {
    request,path:=setupWriteFixture(t,true)
    initialACL:=reviewLinuxACL()
    if err:=unix.Setxattr(path,"system.posix_acl_access",initialACL,0);err!=nil{t.Fatal(err)}
    before,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
    original:=setupWriteRead(t,path)
    plan:=setupWritePlan(t,request)
    revokedACL:=bytes.Clone(initialACL)
    binary.LittleEndian.PutUint16(revokedACL[14:16],0)
    var changed os.FileInfo
    revoke:=func(){
     if err:=unix.Setxattr(path,"system.posix_acl_access",revokedACL,0);err!=nil{t.Fatal(err)}
     changed,err=os.Stat(path);if err!=nil{t.Fatal(err)}
    }
    ops:=defaultSetupWriteOps()
    if during {
     ops.createTemp=func(dir,pattern string)(setupTempFile,error){f,err:=os.CreateTemp(dir,pattern);return &reviewLinuxTemp{f,revoke},err}
    }else{revoke()}
    applyErr:=plan.apply(context.Background(),ops)
    if changed==nil{t.Fatalf("no concurrent change: %v",applyErr)}
    after,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
    afterACL:=reviewLinuxGetXattr(t,path,"system.posix_acl_access")
    if applyErr==nil||!strings.Contains(applyErr.Error(),"changed since setup read"){t.Fatalf("revocation was not rejected: %v",applyErr)}
    if !os.SameFile(before,after)||!bytes.Equal(original,setupWriteRead(t,path))||!bytes.Equal(revokedACL,afterACL){t.Fatal("stale writer replaced source or restored revoked grant")}
    oldStat,newStat:=before.Sys().(*syscall.Stat_t),changed.Sys().(*syscall.Stat_t)
    if oldStat.Ctim==newStat.Ctim {
     collisions++
     if collisions==1{t.Logf("first natural collision: attempt=%d beforeCtim=%+v changedCtim=%+v beforeMode=%o changedMode=%o applyErr=%v initialNamedPerm=%d revokedNamedPerm=%d finalNamedPerm=%d sameInode=%t",attempt,oldStat.Ctim,newStat.Ctim,before.Mode(),changed.Mode(),applyErr,binary.LittleEndian.Uint16(initialACL[14:16]),binary.LittleEndian.Uint16(revokedACL[14:16]),binary.LittleEndian.Uint16(afterACL[14:16]),os.SameFile(before,after))}
    }
   }
   t.Logf("natural same-ctime collisions=%d / 100; every revocation rejected and retained",collisions)
   if collisions==0{t.Fatal("no natural same-ctime collision observed; cannot establish targeted regression")}
  })
 }
}
