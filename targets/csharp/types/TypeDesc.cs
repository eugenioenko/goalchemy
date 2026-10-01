namespace Rt;

/// <summary>A dynamic type: name, equality, key encoding, and method table.</summary>
public sealed class TypeDesc
{
    static int nextId = 1;

    public readonly int Id;
    public readonly string Name;
    public readonly string Kind;
    public readonly Func<object, object, bool> Eq;
    public readonly Func<object, object> Key;
    public readonly Dictionary<string, Fn> Methods;
    public readonly string Basic;
    public readonly bool Comparable;

    public TypeDesc(string name, string kind, Func<object, object, bool> eq, Func<object, object> key,
        Dictionary<string, Fn> methods, string basic, bool comparable)
    {
        Id = System.Threading.Interlocked.Increment(ref nextId);
        Name = name;
        Kind = kind;
        Eq = eq;
        Key = key;
        Methods = methods ?? new Dictionary<string, Fn>();
        Basic = basic;
        Comparable = comparable;
    }

    public static Dictionary<string, Fn> MethodsOf(params object[] kv)
    {
        var m = new Dictionary<string, Fn>();
        for (int i = 0; i + 1 < kv.Length; i += 2) m[(string)kv[i]] = (Fn)kv[i + 1];
        return m;
    }
}
