namespace Rt;

public sealed class ErrorString
{
    public readonly string s;

    public ErrorString(string s) => this.s = s;
}

/// <summary>std.errors.new: a distinct *errors.errorString per call.</summary>
public static partial class R
{
    static class ErrorStringType
    {
        internal static readonly TypeDesc T = new TypeDesc("*errors.errorString", "pointer", (a, b) => a == b,
            a => new IdKey(a), TypeDesc.MethodsOf("Error", new Fn(a => ((ErrorString)a[0]).s)), null, true);
    }

    public static TypeDesc ERRORS_ERROR_STRING => ErrorStringType.T;

    public static Box stdErrorsNew(string text) => new Box(ERRORS_ERROR_STRING, new ErrorString(text));
}
