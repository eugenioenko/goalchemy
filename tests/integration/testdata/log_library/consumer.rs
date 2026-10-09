use goalchemy_generated::*;
use std::sync::{Arc, Mutex};

fn options() -> CallOptions { CallOptions::default() }

fn main() {
    let got: Arc<Mutex<Vec<LogRecord>>> = Arc::new(Mutex::new(Vec::new()));
    let sink = got.clone();
    set_log_handler(Some(Arc::new(move |r: &LogRecord| sink.lock().unwrap().push(r.clone()))), -4);
    assert_eq!(Work(3, options()).wait().unwrap(), 6, "result");
    {
        let got = got.lock().unwrap();
        assert_eq!(got.len(), 4, "record count");
        assert!(got[0].level == -4 && got[0].text == "level=DEBUG msg=start sdk=probe n=3", "debug record");
        assert!(got[1].text == "level=INFO msg=info sdk=probe unicode=\"héllo wörld\"" && got[1].attrs[1].1 == "héllo wörld", "utf8 decoding");
        assert!(got[2].level == 4 && got[2].message == "retry" && got[2].attrs[1] == ("kas.url".to_string(), "https://kas".to_string()), "group attrs");
        let age = std::time::SystemTime::now().duration_since(got[3].time()).unwrap_or_default();
        assert!(got[3].text == "level=ERROR msg=failed err=boom" && age.as_secs() < 60, "error record");
    }
    got.lock().unwrap().clear();
    let sink = got.clone();
    set_log_handler(Some(Arc::new(move |r: &LogRecord| sink.lock().unwrap().push(r.clone()))), 4);
    Work(1, options()).wait().unwrap();
    {
        let got = got.lock().unwrap();
        assert!(got.len() == 2 && got[0].message == "retry" && got[1].level == 8, "host level");
    }
    set_log_handler(Some(Arc::new(|_: &LogRecord| panic!("sink"))), -4);
    assert_eq!(Work(1, options()).wait().unwrap(), 2, "sink panics are discarded");
    set_log_handler(None, 4);
    println!("PASS log library");
}
