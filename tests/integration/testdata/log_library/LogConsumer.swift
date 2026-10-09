import Foundation
import GoalchemyGenerated

private func check(_ condition: Bool, _ message: String) {
    precondition(condition, message)
}

final class Box: @unchecked Sendable {
    var records: [GoalchemyLogRecord] = []
}

@main
struct LogConsumer {
    static func main() async throws {
        let box = Box()
        setLogHandler({ box.records.append($0) }, level: -4)
        check(try Work(3).wait() == 6, "result")
        let got = box.records
        check(got.count == 4, "record count \(got.count)")
        check(got[0].level == -4 && got[0].text == "level=DEBUG msg=start sdk=probe n=3", "debug record")
        check(got[1].text == "level=INFO msg=info sdk=probe unicode=\"héllo wörld\"" && got[1].attrs[1].value == "héllo wörld", "utf8 decoding")
        check(got[2].level == 4 && got[2].message == "retry" && got[2].attrs[1].key == "kas.url" && got[2].attrs[1].value == "https://kas", "group attrs")
        check(got[3].text == "level=ERROR msg=failed err=boom" && abs(got[3].time.timeIntervalSinceNow) < 60, "error record")
        box.records = []
        setLogHandler({ box.records.append($0) }, level: 4)
        _ = try Work(1).wait()
        check(box.records.count == 2 && box.records[0].message == "retry" && box.records[1].level == 8, "host level")
        setLogHandler(nil)
        print("PASS log library")
    }
}
