namespace Rt;

/// <summary>A source panic carrying a Go interface value.</summary>
public sealed class GoPanic : Exception
{
    public readonly Box value;
    public bool recovered;
    public GoPanic prev;

    public GoPanic(Box value) : base("go panic") => this.value = value;
}

/// <summary>An implementation fault: never a source panic.</summary>
public sealed class Fault : Exception
{
    public Fault(string msg) : base("goalchemy fault: " + msg) { }
}
