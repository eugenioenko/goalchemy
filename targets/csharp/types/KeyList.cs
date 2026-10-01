namespace Rt;

/// <summary>A composite map key with structural equality.</summary>
public sealed class KeyList : IEquatable<KeyList>
{
    readonly object[] xs;

    public KeyList(params object[] xs) => this.xs = xs;

    public bool Equals(KeyList o)
    {
        if (o is null || o.xs.Length != xs.Length) return false;
        for (int i = 0; i < xs.Length; i++) if (!Equals(xs[i], o.xs[i])) return false;
        return true;
    }

    public override bool Equals(object o) => o is KeyList k && Equals(k);

    public override int GetHashCode()
    {
        var h = new HashCode();
        foreach (var x in xs) h.Add(x);
        return h.ToHashCode();
    }
}

/// <summary>Identity keys for pointers, channels, and other reference values.</summary>
public sealed class IdKey : IEquatable<IdKey>
{
    readonly object o;

    public IdKey(object o) => this.o = o;

    public bool Equals(IdKey k) => k is not null && ReferenceEquals(o, k.o);
    public override bool Equals(object o) => o is IdKey k && Equals(k);
    public override int GetHashCode() => System.Runtime.CompilerServices.RuntimeHelpers.GetHashCode(o);
}
