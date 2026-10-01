package rt;

/** std.errors.new: errors.New returns a distinct *errors.errorString. */
public final class StdErrorsNew {
    private StdErrorsNew() {}

    static final class ErrorString {
        final String s;

        ErrorString(String s) {
            this.s = s;
        }
    }

    public static final TypeDesc ERRORS_ERROR_STRING = new TypeDesc("*errors.errorString", "pointer", (a, b) -> a == b,
            a -> a, TypeDesc.methods("Error", (Fn) a -> ((ErrorString) a[0]).s), null, true);

    public static Box stdErrorsNew(String text) {
        return new Box(ERRORS_ERROR_STRING, new ErrorString(text));
    }

}
