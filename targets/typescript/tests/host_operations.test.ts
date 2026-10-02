import "../types/node_host.ts";
import { strict as assert } from "node:assert";
import { fromHost } from "../types/utf8.ts";
import { spawnSync } from "node:child_process";
import { writeFileSync, readFileSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { runtimeHost } from "../types/host.ts";
import { Frame, Task, sched, spawn, runMainHost, ret, yieldTask, type HostToken } from "../runtime/task_spawn.ts";
import { Mutex } from "../runtime/std_sync_mutex_lock.ts";
import { stdSyncMutexUnlock } from "../runtime/std_sync_mutex_unlock.ts";
import { plainPanic } from "../types/panic.ts";
import { runPortableSuite } from "./host_operations_suite.ts";
const mode=process.env.GOALCHEMY_TEST_FATAL;
if(mode) {
  class Child extends Frame {
    token!: HostToken;
    step(t: Task): void {
      if(mode==='deadlock') {sched.block(t);return;}
      if(mode==='overflow'){function recurse(): never {return recurse();}recurse();}
      if(this.pc++===0){
        spawn(new class extends Frame {
          step(u: Task): void {
            this.pc++;
            let token:HostToken;
            token=sched.registerHost(u,()=>{
              setTimeout(()=>{writeFileSync(process.env.GOALCHEMY_TEST_MARKER!,'cleaned');token.acknowledgeCleanup();},5);
            });
          }
        }());yieldTask(t);
      }else {if(mode==='mutex')stdSyncMutexUnlock(new Mutex());throw plainPanic('fatal');}
    }
  }
  await runMainHost(new Child(),runtimeHost);
} else {
for(const s of ['\0','\u00ff','é汉','😀','\ud800','\udc00','a\ud800z']) assert.equal(fromHost(s),Buffer.from(s,'utf8').toString('latin1'));
console.log('Node',process.version);
for(const name of await runPortableSuite()) console.log('PASS',name);
console.log('PASS portable UTF8 against Buffer oracle');

mkdirSync('out/typescript-host-operations',{recursive:true});
for(const [mode,text] of [['panic','panic: fatal\n'],['mutex','fatal error: sync: unlock of unlocked mutex\n'],['deadlock','fatal error: all goroutines are asleep - deadlock!\n'],['overflow','runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n']]) {
  const marker=process.cwd()+'/out/typescript-host-operations/node-fatal-cleanup-'+process.pid+'-'+mode;
  rmSync(marker,{force:true});
  const child=spawnSync(process.execPath,[new URL(import.meta.url).pathname],{env:{...process.env,GOALCHEMY_TEST_FATAL:mode,GOALCHEMY_TEST_MARKER:marker},timeout:10000});
  assert.equal(child.status,2,'Node executable exit2 '+mode);assert.equal(child.stderr.toString(),text,'source fatal bytes '+mode);
  if(mode==='panic'||mode==='mutex'){assert.ok(existsSync(marker),'cleanup before real process.exit');assert.equal(readFileSync(marker,'utf8'),'cleaned','exact cleanup marker');}
}
console.log('PASS real Node process.exit2/source panic/deadlock/stack overflow cleanup');
}
