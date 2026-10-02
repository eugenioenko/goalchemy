namespace Rt;

/// <summary>Immutable slice headers over native byte[] or generic object[] storage.</summary>
public sealed class Slice
{
    public readonly Array a;
    public readonly int o, l, c;
    public readonly bool bytes;

    public Slice(Array a, int o, int l, int c) : this(a, o, l, c, a is byte[]) { }
    public Slice(Array a, int o, int l, int c, bool bytes)
    {
        this.a = a; this.o = o; this.l = l; this.c = c; this.bytes = bytes;
    }

    // Scalars crossing generic boundaries retain Go's boxed long representation.
    public object Get(int i) => bytes ? (object)(long)((byte[])a)[o + i] : ((object[])a)[o + i];
    public void Set(int i, object v)
    {
        if (bytes) ((byte[])a)[o + i] = unchecked((byte)(long)v);
        else ((object[])a)[o + i] = v;
    }

    public static readonly Slice NIL = new Slice(null, 0, 0, 0, false);
    public static readonly Slice BYTE_NIL = new Slice(null, 0, 0, 0, true);
}
