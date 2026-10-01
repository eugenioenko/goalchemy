namespace Rt;

/// <summary>core.map.make.</summary>
public static partial class R
{
    public static GoMap makeMap(Func<object, object> keyOf) => new GoMap(keyOf);
}
