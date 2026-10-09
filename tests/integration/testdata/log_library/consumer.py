import importlib
import logging
import sys
import time

sys.path.insert(0, sys.argv[1])
g = importlib.import_module('probe')

got = []
g.set_log_handler(got.append, -4)
assert g.Work(3).result(20) == 6
assert len(got) == 4, got
assert got[0].level == -4 and got[0].text == 'level=DEBUG msg=start sdk=probe n=3', got[0]
assert got[1].text == 'level=INFO msg=info sdk=probe unicode="héllo wörld"' and got[1].attrs[1] == ('unicode', 'héllo wörld'), got[1].attrs
assert got[2].level == 4 and got[2].message == 'retry' and got[2].attrs[1] == ('kas.url', 'https://kas'), got[2].attrs
assert got[3].text == 'level=ERROR msg=failed err=boom' and abs(got[3].time.timestamp() - time.time()) < 60
got.clear()
g.set_log_handler(got.append, 4)
g.Work(1).result(20)
assert [r.message for r in got] == ['retry', 'failed'], got


class Capture(logging.Handler):
    def __init__(self):
        super().__init__()
        self.records = []

    def emit(self, record):
        self.records.append(record)


capture = Capture()
logger = logging.getLogger('goalchemy')
logger.addHandler(capture)
logger.setLevel(logging.INFO)
logger.propagate = False
g.set_log_handler(g.logging_handler(), -4)
g.Work(1).result(20)
assert [(r.levelno, r.getMessage()) for r in capture.records] == [
    (logging.INFO, 'level=INFO msg=info sdk=probe unicode="héllo wörld"'),
    (logging.WARNING, 'level=WARN msg=retry sdk=probe kas.url=https://kas kas.attempt=2'),
    (logging.ERROR, 'level=ERROR msg=failed err=boom')], capture.records
assert capture.records[0].goalchemy.attrs[0] == ('sdk', 'probe')


def failing(record):
    raise RuntimeError('sink')


g.set_log_handler(failing, -4)
assert g.Work(1).result(20) == 2
g.set_log_handler(None)
print('PASS log library')
