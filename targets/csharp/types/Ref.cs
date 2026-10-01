namespace Rt;

/// <summary>A pointer to non-aggregate storage. Pointers to structs and
/// arrays are the stable host objects themselves.</summary>
public abstract class Ref
{
    public abstract object Get();
    public abstract void Set(object v);
}

/// <summary>Heap storage for a boxed variable or new(T).</summary>
public sealed class Cell : Ref
{
    public object v;

    public Cell(object v) => this.v = v;

    public override object Get() => v;
    public override void Set(object x) => v = x;
}

public sealed class FieldRef : Ref
{
    readonly object o;
    readonly Func<object, object> get;
    readonly Action<object, object> set;

    public FieldRef(object o, Func<object, object> get, Action<object, object> set)
    {
        this.o = o;
        this.get = get;
        this.set = set;
    }

    public override object Get() => get(o);
    public override void Set(object v) => set(o, v);
}
