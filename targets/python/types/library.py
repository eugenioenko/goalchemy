"""Owned value-library calls. Source is serialized; publication follows retirement."""
import asyncio
import threading
from concurrent.futures import Future
from .program import HostFault, SourceFatal, reserve, unreserve
from .panic import GoPanic
from .slice import Slice

class LibraryFailure(Exception):
    def __init__(self, kind, fields=None):
        super().__init__('library: ' + kind)
        self.kind = kind
        self.fields = fields or {}

LIMIT = 64 << 20

def library_bool(v):
    if type(v) is not bool: raise LibraryFailure('invalid_argument')
    return v

def library_int(v, bits, signed):
    if type(v) is not int or not (-(1 << (bits-1)) if signed else 0) <= v < (1 << (bits-1) if signed else 1 << bits):
        raise LibraryFailure('invalid_argument')
    return v

def library_string(v):
    if type(v) is not bytes or len(v) > LIMIT: raise LibraryFailure('invalid_argument')
    return v

def library_bytes(v):
    if v is None: return None
    if type(v) not in (bytes, bytearray, memoryview): raise LibraryFailure('invalid_argument')
    try:
        size = v.nbytes if type(v) is memoryview else len(v)
        if size > LIMIT: raise LibraryFailure('invalid_argument')
        return bytearray(v)
    except (ValueError, TypeError):
        raise LibraryFailure('invalid_argument') from None

def library_list(v):
    if type(v) not in (list, tuple) or len(v) > LIMIT: raise LibraryFailure('invalid_argument')
    return v

def library_fields(v, names):
    if type(v) is not dict or any(type(k) is not str or k not in names.split(',') for k in v):
        raise LibraryFailure('invalid_argument')

def library_clear_refs():
    from .ref import _refs
    _refs.clear()

class LibrarySequence:
    """Frame-compatible sequence without running source during construction."""
    def __init__(self, init, call):
        from ..runtime.task_spawn import Frame
        self.pc=0; self.defers=[]; self.parent=None; self.panicking=None
        self.init=init; self.call=call; self.res=[]
    def step(self,t):
        from ..runtime.task_spawn import call, ret
        if self.pc==0:
            self.pc=1; call(t,self.init); return
        if self.pc==1:
            self.pc=2; call(t,self.call()); return
        self.res=t.rv;ret(t,self)
    def results(self): return self.res

_condition = threading.Condition()
_queue = []

class LibraryOperation:
    def __init__(self):
        self._future=Future(); self._cancel=threading.Event(); self._mailbox=None
    def cancel(self):
        if self._future.done(): return False
        self._cancel.set()
        with _condition: _condition.notify_all()
        m=self._mailbox
        if m is not None:
            with m.condition: m.condition.notify_all()
        return True
    def result(self, timeout=None): return self._future.result(timeout)
    def done(self): return self._future.done()
    async def wait(self):
        wrapped=asyncio.wrap_future(self._future)
        try: return await asyncio.shield(wrapped)
        except asyncio.CancelledError:
            self.cancel()
            # Python task cancellation is a request. Wait through repeated requests
            # until native cleanup has actually completed before propagating it.
            while not wrapped.done():
                try: await asyncio.shield(wrapped)
                except asyncio.CancelledError: continue
                except BaseException: break
            if wrapped.done() and not wrapped.cancelled():
                try: wrapped.exception()
                except BaseException: pass
            raise
    def __await__(self): return self.wait().__await__()

def library_submit(factory, output, reset, inputs, options=None):
    from ..runtime.task_spawn import snapshot, _drive, sched
    from ..runtime.std_context_background import std_context_background
    from ..runtime.std_context_with_cancel import std_context_with_cancel
    op=LibraryOperation()
    try:
        owned=snapshot(inputs)
        if options is None: options={}
        if type(options) is not dict or any(k not in ('callbacks',) for k in options): raise LibraryFailure('invalid_argument')
        raw_callbacks=options.get('callbacks', {})
        if type(raw_callbacks) is not dict: raise LibraryFailure('invalid_argument')
        callbacks=dict(raw_callbacks)
        if any(type(k) is not str or not callable(v) for k,v in callbacks.items()): raise LibraryFailure('invalid_argument')
    except LibraryFailure as e:
        op._future.set_exception(e); return op
    except BaseException as e:
        op._future.set_exception(LibraryFailure('invalid_argument')); return op
    with _condition: _queue.append(op)
    def worker():
        failure=None; result=None; acquired=False; state=None
        try:
            with _condition:
                while _queue[0] is not op:
                    if op._cancel.is_set(): raise LibraryFailure('canceled', {'CauseCategory':b'canceled'})
                    _condition.wait()
            if op._cancel.is_set(): raise LibraryFailure('canceled', {'CauseCategory':b'canceled'})
            reserve(); acquired=True
            def start():
                nonlocal state
                s=sched();state=s;s.native_keys={};s.native_next_key=0;s.library_callbacks=callbacks
                op._mailbox=s.mailbox
                ctx,cancel=std_context_with_cancel(std_context_background())
                s.library_poll=lambda: cancel() if op._cancel.is_set() else None
                s.library_poll()
                return factory(ctx,owned)
            def owned_output(rv):
                value=output(rv)
                if op._cancel.is_set(): raise LibraryFailure('canceled', {'CauseCategory':b'canceled'})
                return value
            result=_drive(start,real=True,library=True,convert=owned_output)
        except LibraryFailure as e:
            failure=e.with_traceback(None);failure.__context__=None;failure.__cause__=None
        except GoPanic as e: failure=LibraryFailure('source_panic')
        except SourceFatal as e: failure=LibraryFailure('source_fatal')
        except BaseException:
            failure=LibraryFailure('host_fault')
        finally:
            try:
                if state is not None:
                    state.check();state.native_keys.clear();state.library_callbacks={}
                if acquired: reset()
            except BaseException as e: failure=LibraryFailure('host_fault')
            finally:
                owned.clear();callbacks.clear();op._mailbox=None
                if acquired: unreserve()
                with _condition:
                    _queue.remove(op);_condition.notify_all()
        # Native continuations execute outside reservation and queue locks.
        if failure is not None: op._future.set_exception(failure)
        else: op._future.set_result(result)
    try: threading.Thread(target=worker,name='goalchemy-library',daemon=True).start()
    except BaseException:
        owned.clear();callbacks.clear()
        with _condition: _queue.remove(op);_condition.notify_all()
        op._future.set_exception(LibraryFailure('host_fault'))
    return op
