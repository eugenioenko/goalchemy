namespace Rt;

public static partial class R
{
    // Official Microsoft NuGet API; the host selects vectorization/intrinsics.
    public static long libChecksumCRC32IEEE(Slice data) =>
        (long)System.IO.Hashing.Crc32.HashToUInt32(
            new System.ReadOnlySpan<byte>((byte[])data.a, data.o, data.l));
}
