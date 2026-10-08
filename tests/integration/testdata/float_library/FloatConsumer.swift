import GoalchemyGenerated

private func check(_ condition: Bool, _ message: String) {
    precondition(condition, message)
}

@main
struct FloatConsumer {
    static func main() async throws {
        for value: Double in [1.25, -0.0, .infinity, -.infinity, .nan] {
            let small = try Scalar32(Float(value)).wait()
            let wide = try Scalar64(value).wait()
            check(value.isNaN ? small.isNaN : small.bitPattern == Float(value).bitPattern,
                  "float32 IEEE scalar")
            check(value.isNaN ? wide.isNaN : wide.bitPattern == value.bitPattern,
                  "float64 IEEE scalar")
        }
        check(try Scalar32(Float(16_777_217)).wait() == 16_777_216, "float32 rounding")
        let both = try Both(1.25, -0.0).wait()
        check(both.0 == 1.25 && both.1.bitPattern == (-0.0 as Double).bitPattern,
              "multiple float results")

        let input = Value(Small: 1.25, Wide: -0.0, Values: [2.5],
                          Nested: [[4.5]], Pair: [6.5, 7.5])
        var first = try Echo(input).wait()
        check(first.Small == 1.25 && first.Wide.bitPattern == (-0.0 as Double).bitPattern
              && first.Values![0] == 3.5 && first.Nested![0]![0] == 6.5
              && first.Pair[1] == 7.5, "nested conversion")
        check(input.Values![0] == 2.5 && input.Nested![0]![0] == 4.5, "input ownership")
        first.Values![0] = 99
        first.Nested![0]![0] = 99
        let second = try Echo(input).wait()
        check(second.Values![0] == 3.5 && second.Nested![0]![0] == 6.5, "result ownership")
        let suspended = try await Suspended(input).value()
        check(suspended.Values![0] == 2.5, "async suspension")
        var owned = try Fixed().wait()
        owned.Values![0] = 99
        check(try Fixed().wait().Values![0] == 1.5 && owned.Values![0] == 99,
              "global result ownership")
        let nilValue = try Echo(Value()).wait()
        let emptyValue = try Echo(Value(Values: [], Nested: [])).wait()
        check(nilValue.Values == nil && nilValue.Nested == nil,
              "nil slices stay nil")
        check(emptyValue.Values?.isEmpty == true && emptyValue.Nested?.isEmpty == true,
              "empty slices stay present")

        do {
            _ = try Fail().wait()
            preconditionFailure("source error was lost")
        } catch let failure as GoalchemyFailure {
            check(failure.kind == "source" && failure.fields["Small"] as? Float == 1.25
                  && failure.fields["Wide"] as? Double == -2.5
                  && failure.fields["Values"] as? [Float] == [3.75],
                  "typed source error float fields")
        }
        var invalid = input
        invalid.Pair = [1]
        do {
            _ = try Echo(invalid).wait()
            preconditionFailure("invalid fixed array length was accepted")
        } catch let failure as GoalchemyFailure {
            check(failure.kind == "invalid_argument", "array length validation")
        }
        print("PASS float library")
    }
}
