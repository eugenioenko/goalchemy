//! Owned native library operations. The owner never publishes source objects.
use super::*;
use std::future::Future;
use std::pin::Pin;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Condvar, Mutex};
use std::task::{Context as FutureContext, Poll, Waker};
use std::time::Duration;
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ErrorKind {
    Source,
    Canceled,
    DeadlineExceeded,
    SourcePanic,
    SourceFatal,
    HostFault,
    InvalidArgument,
}
#[derive(Clone, Debug)]
pub struct LibraryError {
    pub kind: ErrorKind,
    pub message: String,
    pub diagnostic: Vec<u8>,
    pub fields: std::collections::BTreeMap<String, HostWire>,
}
impl LibraryError {
    pub fn new(kind: ErrorKind, message: impl Into<String>) -> Self {
        let message = message.into();
        Self {
            kind,
            diagnostic: message.as_bytes().to_vec(),
            message,
            fields: Default::default(),
        }
    }
    pub fn from_bytes(kind: ErrorKind, diagnostic: Vec<u8>) -> Self {
        Self {
            kind,
            message: String::from_utf8_lossy(&diagnostic).into_owned(),
            diagnostic,
            fields: Default::default(),
        }
    }
    pub fn text(&self, name: &str) -> String {
        match self.fields.get(name) {
            Some(HostWire::Bytes(b)) => String::from_utf8_lossy(b).into_owned(),
            _ => String::new(),
        }
    }
    pub fn integer(&self, name: &str) -> i64 {
        match self.fields.get(name) {
            Some(HostWire::Int(i)) => *i,
            _ => 0,
        }
    }
    pub fn normalize_cause(mut self) -> Self {
        self.kind = match self.text("CauseCategory").as_str() {
            "canceled" => ErrorKind::Canceled,
            "deadline_exceeded" => ErrorKind::DeadlineExceeded,
            _ => self.kind,
        };
        self
    }
}
impl std::fmt::Display for LibraryError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{:?}: {}", self.kind, self.message)
    }
}
impl std::error::Error for LibraryError {}
#[derive(Clone, Default)]
pub struct Cancellation(Arc<AtomicBool>);
impl Cancellation {
    pub fn cancel(&self) {
        self.0.store(true, Ordering::Release)
    }
    pub fn is_canceled(&self) -> bool {
        self.0.load(Ordering::Acquire)
    }
}
#[derive(Clone)]
pub struct ProviderRequest {
    pub payload: Vec<u8>,
    pub cancellation: Cancellation,
}
#[derive(Debug)]
pub enum ProviderError {
    Rejected(String),
    Fault(String),
}
/// Return only after all provider resources are released, including cancellation.
pub type Provider =
    Arc<dyn Fn(ProviderRequest) -> Result<Vec<u8>, ProviderError> + Send + Sync + 'static>;
#[derive(Clone, Default)]
pub struct CallOptions {
    pub cancellation: Cancellation,
    pub providers: std::collections::BTreeMap<String, Provider>,
}
struct Completion<T> {
    result: Option<Result<T, LibraryError>>,
    waker: Option<Waker>,
}
struct OperationState<T> {
    completion: Mutex<Completion<T>>,
    wake: Condvar,
    settled: AtomicBool,
}
pub struct Operation<T> {
    state: Arc<OperationState<T>>,
    cancel: Cancellation,
}
impl<T> Operation<T> {
    pub fn cancel(&self) {
        self.cancel.cancel()
    }
    pub fn cancellation(&self) -> Cancellation {
        self.cancel.clone()
    }
    pub fn wait(self) -> Result<T, LibraryError> {
        let mut c = self
            .state
            .completion
            .lock()
            .unwrap_or_else(|e| e.into_inner());
        loop {
            if let Some(r) = c.result.take() {
                return r;
            };
            c = self.state.wake.wait(c).unwrap_or_else(|e| e.into_inner())
        }
    }
}
static LIBRARY_SERIAL: Mutex<()> = Mutex::new(());
impl<T: Send + 'static> Operation<T> {
    pub fn submit(
        options: CallOptions,
        body: impl FnOnce(CallOptions) -> Result<T, LibraryError> + Send + 'static,
    ) -> Self {
        let state = Arc::new(OperationState {
            completion: Mutex::new(Completion {
                result: None,
                waker: None,
            }),
            wake: Condvar::new(),
            settled: AtomicBool::new(false),
        });
        let cancel = options.cancellation.clone();
        let output = state.clone();
        let launch = std::thread::Builder::new()
            .name("goalchemy-library-call".into())
            .spawn(move || {
                let result = (|| {
                    let lock = loop {
                        if options.cancellation.is_canceled() {
                            drop(body);
                            drop(options);
                            return Err(LibraryError::new(
                                ErrorKind::Canceled,
                                "queued call canceled",
                            ));
                        }
                        match LIBRARY_SERIAL.try_lock() {
                            Ok(lock) => break lock,
                            Err(std::sync::TryLockError::WouldBlock) => {
                                std::thread::sleep(Duration::from_millis(2))
                            }
                            Err(_) => {
                                return Err(LibraryError::new(
                                    ErrorKind::HostFault,
                                    "poisoned library reservation",
                                ))
                            }
                        }
                    };
                    let reservation = reserve_entry()
                        .map_err(|e| LibraryError::new(ErrorKind::HostFault, e.0))?;
                    let owner = std::thread::Builder::new()
                        .name("goalchemy-library-owner".into())
                        .stack_size(16 * 1024 * 1024)
                        .spawn(move || body(options));
                    let result = match owner {
                        Ok(h) => h.join().unwrap_or_else(|e| {
                            Err(LibraryError::new(
                                ErrorKind::HostFault,
                                native_fault_message(&e),
                            ))
                        }),
                        Err(e) => Err(LibraryError::new(
                            ErrorKind::HostFault,
                            format!("owner submission: {e}"),
                        )),
                    };
                    drop(reservation);
                    drop(lock);
                    result
                })();
                let waker = {
                    let mut c = output.completion.lock().unwrap_or_else(|e| e.into_inner());
                    c.result = Some(result);
                    output.settled.store(true, Ordering::Release);
                    c.waker.take()
                };
                output.wake.notify_all();
                if let Some(w) = waker {
                    w.wake()
                }
            });
        if let Err(e) = launch {
            state.settled.store(true, Ordering::Release);
            state.completion.lock().unwrap().result = Some(Err(LibraryError::new(
                ErrorKind::HostFault,
                format!("call submission: {e}"),
            )))
        };
        Self { state, cancel }
    }
}
impl<T> Future for Operation<T> {
    type Output = Result<T, LibraryError>;
    fn poll(self: Pin<&mut Self>, cx: &mut FutureContext<'_>) -> Poll<Self::Output> {
        let mut c = self
            .state
            .completion
            .lock()
            .unwrap_or_else(|e| e.into_inner());
        if let Some(r) = c.result.take() {
            Poll::Ready(r)
        } else {
            c.waker = Some(cx.waker().clone());
            Poll::Pending
        }
    }
}
/// Dropping a future requests stop. Use wait/await to observe actual cleanup.
impl<T> Drop for Operation<T> {
    fn drop(&mut self) {
        if !self.state.settled.load(Ordering::Acquire) {
            self.cancel.cancel()
        }
    }
}
pub fn library_bytes(v: Vec<u8>) -> V {
    let n = library_length(v.len());
    V::ByteSlice(byte_array(v).h(), 0, n, n)
}
pub fn library_length(n: usize) -> u32 {
    u32::try_from(n)
        .unwrap_or_else(|_| library_invalid("native collection length exceeds source descriptor"))
}
pub fn library_byte_output(v: &V) -> Vec<u8> {
    let (h, o, n, _, _) = slice_parts(v);
    if h == 0 {
        vec![]
    } else {
        byte_snapshot(h, o as usize, n as usize)
    }
}
pub fn library_strings(v: V) -> Vec<Vec<u8>> {
    let (h, o, n, _, _) = slice_parts(&v);
    (0..n as usize)
        .map(|i| slot(h, o as usize + i).bytes().to_vec())
        .collect()
}
fn sequence_step(t: &Rc<Task>, f: &Rc<Frame>) {
    match f.pc.get() {
        0 => {
            f.pc.set(1);
            call(t, f.l.g(0));
        }
        1 => {
            f.pc.set(2);
            let make = f.prim.borrow_mut().take().unwrap();
            make(t);
        }
        _ => {
            let rv = t.rv.borrow().clone();
            f.l.s(1, tuple(rv));
            ret(t, f);
        }
    }
}
fn sequence_results(f: &Frame) -> Vec<V> {
    match f.l.g(1) {
        V::Tuple(x) => x.to_vec(),
        _ => vec![],
    }
}
pub fn library_sequence(init: V, entry: impl FnOnce() -> V + 'static) -> V {
    let f = Frame::new(2, sequence_step, Some(sequence_results));
    f.l.s(0, init);
    f.prim.replace(Some(Box::new(move |t| call(t, entry()))));
    V::Frame(f)
}
thread_local! {pub static LIBRARY_PROVIDERS:RefCell<std::collections::BTreeMap<String,Provider>>=RefCell::new(Default::default());}
pub fn library_owner<T>(
    globals: usize,
    zero: fn(),
    options: CallOptions,
    factory: impl FnOnce(V) -> V,
    output: impl FnOnce(Vec<V>) -> Result<T, LibraryError>,
) -> Result<T, LibraryError> {
    let mut options = Some(options);
    let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        library_install();
        init_globals(globals);
        zero();
        let c = new_child(background());
        let _context_root = temp_root(&[c.clone()]);
        let options = options.take().unwrap();
        LIBRARY_PROVIDERS.with(|p| *p.borrow_mut() = options.providers);
        let cancel = options.cancellation;
        let publication_cancel = cancel.clone();
        let ctx = c.clone();
        set_host_poll(Some(Rc::new(move || {
            if cancel.is_canceled() {
                cancel_ctx(ctx.clone(), context_canceled())
            }
        })));
        let frame = factory(c);
        let main = sched(|s| s.cur.clone());
        main.frame.replace(Some(frame.frame()));
        ready(&main);
        library_run(&main);
        let rv = main.rv.borrow().clone();
        let output = output(rv);
        if publication_cancel.is_canceled() {
            match output {
                Err(e) if e.kind == ErrorKind::HostFault => Err(e),
                Err(e) if e.kind == ErrorKind::Canceled => Err(e),
                _ => Err(LibraryError::new(
                    ErrorKind::Canceled,
                    "active call canceled",
                )),
            }
        } else {
            output
        }
    }))
    .map_err(library_classify)
    .and_then(|r| r);
    set_host_poll(None);
    let retirement = retire_owner();
    LIBRARY_PROVIDERS.with(|p| p.borrow_mut().clear());
    #[cfg(feature = "native")]
    crypto_retire();
    match retirement {
        Err(e) => Err(LibraryError::new(ErrorKind::HostFault, e)),
        Ok(()) => result,
    }
}

fn library_classify(e: Box<dyn std::any::Any + Send>) -> LibraryError {
    if let Some(e) = e.downcast_ref::<LibraryInvalid>() {
        return LibraryError::new(ErrorKind::InvalidArgument, e.0.clone());
    };
    match classify(e) {
        HostError::Fault(s) => LibraryError::new(ErrorKind::HostFault, s),
        HostError::Fatal(s) => LibraryError::new(ErrorKind::SourceFatal, s),
        HostError::SourcePanic(b) => LibraryError::from_bytes(ErrorKind::SourcePanic, b),
        HostError::Blocked => LibraryError::new(ErrorKind::SourceFatal, "blocked source"),
    }
}

struct LibraryInvalid(String);
pub fn library_invalid(s: &str) -> ! {
    std::panic::resume_unwind(Box::new(LibraryInvalid(s.into())))
}
