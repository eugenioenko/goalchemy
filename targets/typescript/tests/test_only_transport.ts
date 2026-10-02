// TEST ONLY: exercises pending Promise/fetch I/O. This deliberately supplies no
// production lib.http.do mapping, shared response bounds, header fidelity,
// redirect policy, credential or crypto contract.
import { sched, type Task } from '../runtime/task_spawn.ts';
export function testOnlyFetch(t: Task, url: string, body: Uint8Array): void {
  const copied=body.slice();
  const controller=new AbortController();
  const token=sched.registerHost(t,()=>controller.abort());
  sched.launchHost(token,async()=>{
    try {
      const response=await fetch(url,{method:'POST',body:copied,signal:controller.signal});
      return [new Uint8Array(await response.arrayBuffer())];
    } catch(e) { return [{testOnlyTransportError:String(e)}]; }
  });
}
