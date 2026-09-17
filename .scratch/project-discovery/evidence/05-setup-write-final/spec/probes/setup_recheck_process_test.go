package discovery

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "testing"
  "time"
)

type reviewProcessResult struct {PID int; Personal bool; Error string}
func reviewWaitPath(path string)error{
  until:=time.Now().Add(5*time.Second)
  for time.Now().Before(until){if _,err:=os.Stat(path);err==nil{return nil};time.Sleep(2*time.Millisecond)}
  return fmt.Errorf("timeout waiting for %s",path)
}
func TestReviewRecheckProcessHelper(t *testing.T){
  role:=os.Getenv("CTX_RECHECK_ROLE")
  if role==""{t.Skip("subprocess helper")}
  root:=os.Getenv("CTX_RECHECK_ROOT")
  request:=SetupRequest{Cwd:filepath.Join(root,"checkout"),Home:filepath.Join(root,"home"),Records:filepath.Join(root,"records"),Personal:role=="personal"}
  if request.Personal{request.AllowSources=[]string{request.Cwd}}
  plan,err:=PrepareSetup(context.Background(),request);if err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(filepath.Join(root,role+"-prepared"),nil,0600);err!=nil{t.Fatal(err)}
  if err:=reviewWaitPath(filepath.Join(root,"start-"+role));err!=nil{t.Fatal(err)}
  ops:=defaultSetupWriteOps()
  ops.rename=func(from,to string)error{
    if err:=os.WriteFile(filepath.Join(root,role+"-commit"),nil,0600);err!=nil{return err}
    if err:=reviewWaitPath(filepath.Join(root,"release"));err!=nil{return err}
    return os.Rename(from,to)
  }
  err=plan.apply(context.Background(),ops)
  result:=reviewProcessResult{PID:os.Getpid(),Personal:request.Personal}
  if err!=nil{result.Error=err.Error()}
  data,_:=json.Marshal(result)
  if err:=os.WriteFile(filepath.Join(root,role+"-result.json"),data,0600);err!=nil{t.Fatal(err)}
}

func TestReviewRecheckSeparateProcesses(t *testing.T){
  for _,first:=range []string{"shared","personal"}{
    t.Run(first+"-first",func(t *testing.T){
      request,_:=setupWriteFixture(t,false);root:=filepath.Dir(request.Home)
      second:="personal";if first=="personal"{second="shared"}
      done:=make(chan error,2)
      for _,role:=range []string{"shared","personal"}{
        go func(role string){
          cmd:=exec.Command(os.Args[0],"-test.run=^TestReviewRecheckProcessHelper$","-test.v")
          cmd.Env=append(os.Environ(),"CTX_RECHECK_ROLE="+role,"CTX_RECHECK_ROOT="+root)
          data,err:=cmd.CombinedOutput();if err!=nil{err=fmt.Errorf("%s: %w: %s",role,err,data)};done<-err
        }(role)
      }
      touch:=func(name string){if err:=os.WriteFile(filepath.Join(root,name),nil,0600);err!=nil{t.Fatal(err)}}
      for _,role:=range []string{"shared","personal"}{if err:=reviewWaitPath(filepath.Join(root,role+"-prepared"));err!=nil{t.Fatal(err)}}
      touch("start-"+first)
      if err:=reviewWaitPath(filepath.Join(root,first+"-commit"));err!=nil{t.Fatal(err)}
      touch("start-"+second)
      time.Sleep(200*time.Millisecond)
      _,secondReachedCommit:=os.Stat(filepath.Join(root,second+"-commit"))
      touch("release")
      for range 2{if err:=<-done;err!=nil{t.Fatal(err)}}
      var results []reviewProcessResult
      for _,role:=range []string{first,second}{
        data,err:=os.ReadFile(filepath.Join(root,role+"-result.json"));if err!=nil{t.Fatal(err)}
        var result reviewProcessResult;if err:=json.Unmarshal(data,&result);err!=nil{t.Fatal(err)};results=append(results,result)
      }
      t.Logf("separate-process results: %+v",results)
      if secondReachedCommit==nil || results[0].Error!="" || !strings.Contains(results[1].Error,"conflicting") || results[0].PID==results[1].PID {
        t.Fatal("want one commit and one conflict across serialized independent processes")
      }
      if _,err:=Resolve(context.Background(),Request{Cwd:request.Cwd,Home:request.Home});err!=nil{t.Fatal(err)}
    })
  }
}
