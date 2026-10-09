import * as g from './main.ts';

function check(value: boolean, label: string): void {
  if (!value) throw new Error(label);
}

let got: g.LogRecord[] = [];
g.setLogHandler(r => { got.push(r); }, -4);
check(await g.Work(3n) === 6n, 'result');
check(got.length === 4, 'record count ' + got.length);
check(got[0]!.level === -4 && got[0]!.text === 'level=DEBUG msg=start sdk=probe n=3', 'debug record');
check(got[1]!.text === 'level=INFO msg=info sdk=probe unicode="héllo wörld"' && got[1]!.attrs[1]![1] === 'héllo wörld', 'utf8 decoding');
check(got[2]!.level === 4 && got[2]!.message === 'retry' && got[2]!.attrs[1]![0] === 'kas.url' && got[2]!.attrs[1]![1] === 'https://kas', 'group attrs');
check(got[3]!.text === 'level=ERROR msg=failed err=boom' && Math.abs(got[3]!.time.getTime() - Date.now()) < 60000, 'error record');
got = [];
g.setLogHandler(r => { got.push(r); }, 4);
await g.Work(1n);
check(got.length === 2 && got[0]!.message === 'retry' && got[1]!.level === 8, 'host level');
g.setLogHandler(r => { throw new Error('sink'); }, -4);
check(await g.Work(1n) === 2n, 'sink errors are discarded');
g.setLogHandler(null);
console.log('PASS log library');
