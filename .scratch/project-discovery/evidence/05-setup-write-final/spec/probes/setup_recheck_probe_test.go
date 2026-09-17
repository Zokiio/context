package discovery

import (
    "bytes"
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "syscall"
    "testing"
    "time"

    "golang.org/x/sys/unix"
)

func reviewRecheckPlans(t *testing.T, firstPersonal, agree bool) (*SetupPlan,*SetupPlan,SetupRequest) {
    t.Helper()
    request,_:=setupWriteFixture(t,false)
    request.AllowSources=nil
    other:=request
    other.Personal=true
    if !agree { other.AllowSources=[]string{request.Cwd} }
    a,b:=setupWritePlan(t,request),setupWritePlan(t,other)
    if firstPersonal { a,b=b,a }
    return a,b,request
}

func TestReviewRecheckSerializesSharedPersonal(t *testing.T) {
    for _, firstPersonal:=range []bool{false,true} {
        t.Run(fmt.Sprintf("personal-first=%t",firstPersonal),func(t *testing.T){
            first,second,request:=reviewRecheckPlans(t,firstPersonal,false)
            entered:=make(chan string,2)
            release:=make(chan struct{})
            defer close(release)
            results:=make(chan error,2)
            run:=func(plan *SetupPlan,label string){
                ops:=defaultSetupWriteOps()
                ops.rename=func(from,to string)error{ entered<-label; <-release; return os.Rename(from,to) }
                results<-plan.apply(context.Background(),ops)
            }
            go run(first,"first")
            select { case <-entered: case <-time.After(5*time.Second): t.Fatal("first writer did not reach commit") }
            go run(second,"second")
            concurrent:=false
            select {case <-entered: concurrent=true; case <-time.After(200*time.Millisecond):}
            release<-struct{}{}
            if concurrent { release<-struct{}{} }
            firstErr:=<-results
            var secondErr error
            select {case secondErr=<-results:case <-time.After(5*time.Second):t.Fatal("second writer stalled")}
            scope,err:=Resolve(context.Background(),Request{Cwd:request.Cwd,Home:request.Home})
            t.Logf("firstPersonal=%t errors=%v / %v discovery=%v",firstPersonal,firstErr,secondErr,err)
            if concurrent || (firstErr==nil)==(secondErr==nil) || err!=nil || scope.Project==nil {
                t.Fatalf("want one commit, one rejection, and valid discovery; concurrent=%t",concurrent)
            }
            rejected:=firstErr
            if rejected==nil {rejected=secondErr}
            if !strings.Contains(rejected.Error(),"conflicting") {t.Fatalf("wrong rejection: %v",rejected)}
        })
    }
}

func TestReviewRecheckAllowsMatchingDeclarations(t *testing.T) {
    first,second,request:=reviewRecheckPlans(t,false,true)
    if err:=first.Apply(context.Background());err!=nil{t.Fatal(err)}
    if err:=second.Apply(context.Background());err!=nil{t.Fatal(err)}
    if _,err:=Resolve(context.Background(),Request{Cwd:request.Cwd,Home:request.Home});err!=nil{t.Fatal(err)}
}

func reviewRecheckACL(t *testing.T,path string)string{
    t.Helper()
    data,err:=exec.Command("ls","-le",path).CombinedOutput()
    if err!=nil{t.Fatalf("inspect ACL: %v: %s",err,data)}
    _,acl,_:=strings.Cut(string(data),"\n")
    return acl
}
func reviewRecheckChmodACL(t *testing.T,path,acl string){
    t.Helper()
    data,err:=exec.Command("chmod","+a",acl,path).CombinedOutput()
    if err!=nil{t.Fatalf("set ACL: %v: %s",err,data)}
}
func reviewRecheckOtherGroup(t *testing.T,path string)int{
    t.Helper()
    info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
    old:=int(info.Sys().(*syscall.Stat_t).Gid)
    groups,err:=os.Getgroups();if err!=nil{t.Fatal(err)}
    for _,g:=range groups {if g!=old {if err:=os.Chown(path,-1,g);err==nil{return g}}}
    t.Skip("no permitted alternative group")
    return old
}

func TestReviewRecheckPreservesOwnershipACLAndMode(t *testing.T){
    request,path:=setupWriteFixture(t,true)
    wantGID:=reviewRecheckOtherGroup(t,path)
    reviewRecheckChmodACL(t,path,"everyone deny write")
    acl:=reviewRecheckACL(t,path)
    before,_:=os.Stat(path)
    plan:=setupWritePlan(t,request)
    if err:=plan.Apply(context.Background());err!=nil{t.Fatal(err)}
    after,_:=os.Stat(path)
    old,new:=before.Sys().(*syscall.Stat_t),after.Sys().(*syscall.Stat_t)
    if new.Uid!=old.Uid || int(new.Gid)!=wantGID || after.Mode()!=before.Mode() || reviewRecheckACL(t,path)!=acl {
        t.Fatalf("permissions changed: before=%#v after=%#v ACL=%q",old,new,reviewRecheckACL(t,path))
    }
}

func TestReviewRecheckClearsInheritedACL(t *testing.T){
    request,path:=setupWriteFixture(t,true)
    reviewRecheckChmodACL(t,filepath.Dir(path),"everyone allow read,file_inherit,directory_inherit")
    before:=reviewRecheckACL(t,path)
    if before!=""{t.Fatalf("fixture source inherited ACL unexpectedly: %q",before)}
    plan:=setupWritePlan(t,request)
    if err:=plan.Apply(context.Background());err!=nil{t.Fatal(err)}
    if got:=reviewRecheckACL(t,path);got!=before{t.Fatalf("temporary inherited ACL widened source access: %q",got)}
}

type reviewRecheckFile struct{*os.File; change func()}
func(f *reviewRecheckFile)Sync()error{f.change();return f.File.Sync()}

func TestReviewRecheckDetectsStaleACLAndGroup(t *testing.T){
    for _,kind:=range []string{"acl","group"}{
      for _,duringWrite:=range []bool{false,true}{
        t.Run(fmt.Sprintf("%s-during-write=%t",kind,duringWrite),func(t *testing.T){
          request,path:=setupWriteFixture(t,true)
          original:=setupWriteRead(t,path)
          plan:=setupWritePlan(t,request)
          changedACL:=""
          changedGID:=uint32(0)
          change:=func(){
            if kind=="acl"{reviewRecheckChmodACL(t,path,"everyone deny write");changedACL=reviewRecheckACL(t,path)}else{changedGID=uint32(reviewRecheckOtherGroup(t,path))}
          }
          ops:=defaultSetupWriteOps()
          if duringWrite{
            ops.createTemp=func(dir,pattern string)(setupTempFile,error){f,err:=os.CreateTemp(dir,pattern);return &reviewRecheckFile{f,change},err}
          }else{change()}
          err:=plan.apply(context.Background(),ops)
          if err==nil||!strings.Contains(err.Error(),"changed since setup read"){t.Fatalf("concurrent %s was ignored: %v",kind,err)}
          after,_:=os.Stat(path)
          if !bytes.Equal(original,setupWriteRead(t,path)){t.Fatal("stale plan changed file contents")}
          if kind=="acl"&&reviewRecheckACL(t,path)!=changedACL{t.Fatal("stale plan stripped ACL")}
          if kind=="group"&&after.Sys().(*syscall.Stat_t).Gid!=changedGID{t.Fatal("stale plan reset group")}
        })
      }
    }
}

type reviewRecheckInfo struct{os.FileInfo; stat syscall.Stat_t}
func(i reviewRecheckInfo)Sys()any{return &i.stat}
func TestReviewRecheckDetectsUIDWithoutPrivilege(t *testing.T){
    _,path:=setupWriteFixture(t,true)
    info,_:=os.Stat(path)
    changed:=*info.Sys().(*syscall.Stat_t)
    changed.Uid++
    if sameSetupPermissions(info,reviewRecheckInfo{info,changed}) {t.Fatal("sameSetupPermissions ignores uid change")}
}

func TestReviewRecheckFailsSafelyOnProtectionFlag(t *testing.T){
    request,path:=setupWriteFixture(t,true)
    if err:=syscall.Chflags(path,unix.UF_IMMUTABLE);err!=nil{t.Skipf("cannot set immutable flag: %v",err)}
    defer syscall.Chflags(path,0)
    original:=setupWriteRead(t,path)
    before,_:=os.Stat(path)
    plan:=setupWritePlan(t,request)
    if err:=plan.Apply(context.Background());err==nil{t.Fatal("protection flag was silently lost")}
    after,_:=os.Stat(path)
    if !os.SameFile(before,after)||!bytes.Equal(original,setupWriteRead(t,path))||after.Sys().(*syscall.Stat_t).Flags&unix.UF_IMMUTABLE==0{t.Fatal("failed setup stripped protected file")}
}

func TestReviewRecheckFailsSafelyWhenACLUnreadable(t *testing.T){
    request,path:=setupWriteFixture(t,true)
    original:=setupWriteRead(t,path)
    before,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
    reviewRecheckChmodACL(t,path,"everyone deny readsecurity")
    defer exec.Command("chmod","-N",path).Run()
    plan,err:=PrepareSetup(context.Background(),request)
    if err==nil {err=plan.Apply(context.Background())}
    if err==nil{t.Fatal("expected safe failure with unreadable security metadata")}
    if _,statErr:=os.Lstat(path);!os.IsPermission(statErr){t.Fatalf("original denial disappeared: %v",statErr)}
    if data,removeErr:=exec.Command("chmod","-N",path).CombinedOutput();removeErr!=nil{t.Fatalf("cannot clear test ACL after safe failure: %v: %s",removeErr,data)}
    after,statErr:=os.Stat(path);if statErr!=nil{t.Fatal(statErr)}
    if !os.SameFile(before,after)||!bytes.Equal(original,setupWriteRead(t,path)){t.Fatalf("failed ACL access changed file: %v",err)}
    t.Logf("safe permission error: %v",err)
}
