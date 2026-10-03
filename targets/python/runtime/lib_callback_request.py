"""Native provider worker. Async cancellation awaits provider finally/release."""
import asyncio
import threading
from concurrent.futures import Future
from .task_spawn import sched, register_host, fault_text
from .std_context_err import check_context
from .std_errors_new import std_errors_new
from .lib_crypto_close import byte_input, slice_bytes, Reject
from ..types.slice import BYTE_NIL
class ProviderRejected(Exception): pass
class ProviderRequest:
    def __init__(self,event,payload):self.cancellation=event;self.payload=payload
    @property
    def canceled(self):return self.cancellation.is_set()
async def _await_provider(value,event):
    task=asyncio.ensure_future(value);cancelled=False
    while not task.done():
        if event.is_set() and not cancelled:task.cancel();cancelled=True
        await asyncio.wait([task],timeout=0.05)
    return await task

def lib_callback_request(t,context,name,payload):
    s=sched();s.check()
    if context is None or type(name) is not bytes:
        t.rv=[BYTE_NIL,std_errors_new(b'callback: invalid input')];return
    check_context(context)
    if context.err is not None:t.rv=[BYTE_NIL,context.err];return
    try:
        text=name.decode('utf-8','strict');body=byte_input(payload)
        if not text or len(name)>256:raise Reject('invalid')
    except (Reject,UnicodeError):t.rv=[BYTE_NIL,std_errors_new(b'callback: invalid input')];return
    provider=getattr(s,'library_callbacks',{}).get(text)
    if provider is None:t.rv=[BYTE_NIL,std_errors_new(b'callback: provider not found')];return
    event=threading.Event()
    def decode(w):return [BYTE_NIL,std_errors_new(w[0])] if w[0] is not None else [slice_bytes(w[1]),None]
    token=register_host(t,context,event.set,decode=decode,cancel_result=lambda err:[BYTE_NIL,err])
    def work():
        nonlocal body,provider
        fault=None;record=None;value=None
        try:
            value=provider(ProviderRequest(event,body))
            if asyncio.iscoroutine(value):value=asyncio.run(_await_provider(value,event))
            elif type(value) is Future:
                # A canceled Future does not acknowledge user resource cleanup;
                # provider Future contract requires settlement after its cleanup.
                value=value.result()
            if type(value) is not bytes or len(value)>128*1024:raise ProviderRejected('provider: invalid response')
            record=[None,value]
        except (ProviderRejected,asyncio.CancelledError):record=[b'provider: rejected',b'']
        except BaseException as e:fault='provider fault: '+fault_text(e)
        finally:value=None;body=None;provider=None
        token.publish(record,fault=fault);token.acknowledge_cleanup()
    try:threading.Thread(target=work,name='goalchemy-provider',daemon=True).start()
    except BaseException as e:token.publish(fault='provider submission: '+fault_text(e));token.acknowledge_cleanup()
