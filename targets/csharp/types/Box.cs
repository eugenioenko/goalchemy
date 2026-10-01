namespace Rt;

/// <summary>A non-nil interface value: dynamic type and value.</summary>
public sealed class Box
{
    public readonly TypeDesc t;
    public readonly object v;

    public Box(TypeDesc t, object v)
    {
        this.t = t;
        this.v = v;
    }

    public static Box box(TypeDesc t, object v) => new Box(t, v);

    public static bool ifaceEq(Box a, Box b)
    {
        if (a == null || b == null) return a == b;
        if (a.t != b.t) return false;
        return a.t.Eq(a.v, b.v);
    }

    public static object ifaceKey(Box a)
    {
        if (a == null) return "nil";
        return new KeyList(a.t.Id, a.t.Key(a.v));
    }

    public static string implementsAll(TypeDesc t, params string[] ids)
    {
        foreach (var id in ids) if (!t.Methods.ContainsKey(id)) return id;
        return null;
    }
}
