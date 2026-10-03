namespace Rt;

public static partial class R
{
    // CLR type initialization publishes immutable tables safely to all callers.
    private static class CRC32IEEETables
    {
        internal static readonly uint[] Values = Build();

        private static uint[] Build()
        {
            var table = new uint[8 * 256];
            for (int i = 0; i < 256; i++)
            {
                uint crc = (uint)i;
                for (int bit = 0; bit < 8; bit++)
                    crc = (crc >> 1) ^ ((crc & 1) != 0 ? 0xedb88320u : 0u);
                table[i] = crc;
            }
            for (int row = 1; row < 8; row++)
                for (int i = 0; i < 256; i++)
                {
                    uint crc = table[(row - 1) * 256 + i];
                    table[row * 256 + i] = (crc >> 8) ^ table[crc & 255];
                }
            return table;
        }
    }

    public static long libChecksumCRC32IEEE(Slice data)
    {
        var table = CRC32IEEETables.Values;
        var bytes = (byte[])data.a;
        uint crc = 0xffffffffu;
        int i = data.o, end = i + data.l;
        for (; i + 8 <= end; i += 8)
        {
            crc ^= (uint)bytes[i] | ((uint)bytes[i + 1] << 8) |
                ((uint)bytes[i + 2] << 16) | ((uint)bytes[i + 3] << 24);
            crc = table[1792 + (crc & 255)] ^ table[1536 + ((crc >> 8) & 255)] ^
                table[1280 + ((crc >> 16) & 255)] ^ table[1024 + (crc >> 24)] ^
                table[768 + bytes[i + 4]] ^ table[512 + bytes[i + 5]] ^
                table[256 + bytes[i + 6]] ^ table[bytes[i + 7]];
        }
        for (; i < end; i++) crc = (crc >> 8) ^ table[(crc ^ bytes[i]) & 255];
        return (long)~crc;
    }
}
